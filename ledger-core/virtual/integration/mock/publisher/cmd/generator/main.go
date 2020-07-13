// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path"
	"sort"
	"strings"
	"text/template"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var (
	ProjectPath    = "github.com/insolar/assured-ledger/ledger-core"
	HandlersSubDir = "virtual/handlers"
	MockPath       = "virtual/integration/mock/publisher/checker/typed.go"
)

func GetRealProjectDir() (string, error) {
	goPath := build.Default.GOPATH
	if goPath == "" {
		return "", errors.New("GOPATH is not set")
	}
	contractsPath := ""
	for _, p := range strings.Split(goPath, ":") {
		contractsPath = path.Join(p, "src", ProjectPath)
		_, err := os.Stat(contractsPath)
		if err == nil {
			return contractsPath, nil
		}
	}
	return "", throw.New("Can't find project dir in GOPATH")
}

func getTypeOfExpr(expr ast.Expr) (x, sel string, star bool) {
	switch arg := expr.(type) {
	case *ast.StarExpr:
		star = true
		expr = arg.X
	case *ast.SelectorExpr:
	default:
		return "", "", false
	}
	x, sel = getSelectorOfExpr(expr)
	return
}

func getSelectorOfExpr(expr ast.Expr) (x, sel string) {
	switch arg := expr.(type) {
	case *ast.SelectorExpr:
		if arg.Sel != nil {
			sel = arg.Sel.Name
		}
		if id, ok := arg.X.(*ast.Ident); ok {
			x = id.Name
		}
	case *ast.Ident:
		return "", arg.Name
	}
	return
}

func getReceiver(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) != 1 {
		return ""
	}
	id := recv.List[0]
	switch len(id.Names) {
	case 0:
	case 1:
		break
	default:
		return ""
	}

	switch x, sel, _ := getTypeOfExpr(id.Type); {
	case sel == "":
		return ""
	case x != "":
		return ""
	default:
		return sel
	}
}

func checkArgumentType(paramList *ast.FieldList, position int, tp string) bool {
	if paramList.NumFields() <= position {
		return false
	}

	typeAst := paramList.List[position].Type
	typeSelector, ok := typeAst.(*ast.SelectorExpr)
	if !ok {
		return false
	} else if typeSelector.Sel.Name != tp {
		return false
	} // todo: check that imports are right here

	return true
}

func loadFiles(files []*ast.File) []string {
	handlers := make([]string, 0)

	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			switch {
			case !ok:
				continue
			case getReceiver(fn.Recv) == "":
				continue
			case fn.Name.Name != "GetStateMachineDeclaration":
				continue
			case fn.Type.Params.NumFields() != 0:
				continue
			case fn.Type.Results.NumFields() != 1:
				continue
			case !checkArgumentType(fn.Type.Results, 0, "StateMachineDeclaration"):
				continue

			default:
				msgName := strings.TrimPrefix(getReceiver(fn.Recv), "SM")
				handlers = append(handlers, msgName)
			}
		}
	}

	return handlers
}

func parseHandlers() ([]string, error) {
	projectPath, err := GetRealProjectDir()
	if err != nil {
		return nil, err
	}
	handlersPath := path.Join(projectPath, HandlersSubDir)

	fileFilter := func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test")
	}

	set := token.NewFileSet()
	pkgs, err := parser.ParseDir(set, handlersPath, fileFilter, 0)
	if err != nil {
		return nil, throw.W(err, "failed to parse package")
	}

	handlerFiles := make([]*ast.File, 0)
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			handlerFiles = append(handlerFiles, file)
		}
	}
	messages := loadFiles(handlerFiles)

	sort.Slice(messages, func(i, j int) bool { return strings.Compare(messages[i], messages[j]) < 0 })

	return messages, nil
}

func generateMock(messages []string) (*bytes.Buffer, error) {
	data := struct {
		Messages []string
	}{
		Messages: messages,
	}
	var (
		tpl = template.New("typed.go.tpl")
		buf = bytes.Buffer{}
	)

	_, err := tpl.Parse(PublishCheckerMock)
	if err != nil {
		return nil, throw.W(err, "failed to parse template")
	}

	err = tpl.ExecuteTemplate(&buf, "typed.go.tpl", data)
	if err != nil {
		return nil, throw.W(err, "failed to execute template")
	}

	return &buf, nil
}

func processMock(mock *bytes.Buffer, output string) error {
	code, err := format.Source(mock.Bytes())
	if err != nil {
		fmt.Println(err)
		errPrefix := "couldn't format code: "
		lines := strings.Split(mock.String(), "\n")
		for lineNo, line := range lines {
			errPrefix += fmt.Sprintf("\n%04d | %s", lineNo, line)
		}
		return throw.W(err, errPrefix)
	}

	targetFile, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return throw.W(err, "failed to open file for write")
	}

	_, err = targetFile.Write(code)
	if err != nil {
		return throw.W(err, "failed to write code")
	}

	return nil
}

func main() {
	messages, err := parseHandlers()
	if err != nil {
		panic(err)
	}

	mock, err := generateMock(messages)
	if err != nil {
		panic(err)
	}

	projectDir, err := GetRealProjectDir()
	if err != nil {
		panic(err)
	}
	output := path.Join(projectDir, MockPath)

	if err := processMock(mock, output); err != nil {
		panic(err)
	}
}
