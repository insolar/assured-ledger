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

func getTypeName(typeAst ast.Expr, withPackage bool) string {
	typeSelector, ok := typeAst.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	if packageName, ok := typeSelector.X.(*ast.Ident); withPackage && ok {
		return packageName.Name + "." + typeSelector.Sel.Name
	}

	return typeSelector.Sel.Name
}

func getStarTypeName(typeAst ast.Expr, withPackage bool) string {
	typeStarExpr, ok := typeAst.(*ast.StarExpr)
	if !ok {
		return ""
	}

	return getTypeName(typeStarExpr.X, withPackage)
}

func getOptStarTypeName(typeAst ast.Expr, withPackage bool) string {
	if rv := getStarTypeName(typeAst, withPackage); rv != "" {
		return rv
	}
	return getTypeName(typeAst, withPackage)
}

func checkArgumentType(paramList *ast.FieldList, position int, tp string) bool {
	if paramList.NumFields() <= position {
		return false
	}

	return getTypeName(paramList.List[position].Type, false) == tp
}

// GetStateMachineName returns empty string on parsing non GetStateMachineDeclaration
func GetStateMachineName(fn *ast.FuncDecl) string {
	switch {
	case getReceiver(fn.Recv) == "":
		return ""
	case fn.Name.Name != "GetStateMachineDeclaration":
		return ""
	case fn.Type.Params.NumFields() != 0:
		return ""
	case fn.Type.Results.NumFields() != 1:
		return ""
	case !checkArgumentType(fn.Type.Results, 0, "StateMachineDeclaration"):
		return ""

	default:
		return getReceiver(fn.Recv)
	}
}

func getStructName(fn *ast.TypeSpec) (string, string) {
	structDef, ok := fn.Type.(*ast.StructType)
	if fn.Name.Name == "" || !ok {
		return "", ""
	}

	for _, val := range structDef.Fields.List {
		if len(val.Names) == 1 && val.Names[0].Name == "Payload" {
			return fn.Name.Name, getOptStarTypeName(val.Type, true)
		}
	}

	return "", ""
}

func loadFiles(files []*ast.File) []string {
	var (
		smList         = make(map[string]struct{})
		payloadMapping = make(map[string]string)
	)

	for _, file := range files {
		for _, decl := range file.Decls {

			// extract all declarations of
			if fn, ok := decl.(*ast.FuncDecl); ok {
				smList[GetStateMachineName(fn)] = struct{}{}
			} else if general, ok := decl.(*ast.GenDecl); ok && general.Tok == token.TYPE {
				for _, rawSpec := range general.Specs {
					spec, ok := rawSpec.(*ast.TypeSpec)
					if !ok {
						panic(throw.IllegalState())
					}

					smName, payloadType := getStructName(spec)
					switch {
					case smName == "" || payloadType == "":
						// do nothing
					case !strings.HasPrefix(payloadType, "payload.") && !strings.HasPrefix(payloadType, "rms."):
						// do nothing
					default:
						payloadMapping[smName] = payloadType
					}
				}
			}
		}
	}

	messageMap := make(map[string]struct{})

	for stateMachineName := range smList {
		if stateMachineName == "" {
			continue
		}

		messageName, ok := payloadMapping[stateMachineName]
		if !ok {
			continue
		} else if messageName == "" {
			continue
		}

		messageMap[messageName] = struct{}{}
	}

	messages := make([]string, 0, len(messageMap))
	for messageName := range messageMap {
		messages = append(messages, messageName)
	}

	sort.Slice(messages, func(i, j int) bool { return strings.Compare(messages[i], messages[j]) < 0 })

	return messages
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

	return messages, nil
}

func checkRMSIsUsed(messages []string) bool {
	for _, message := range messages {
		if strings.HasPrefix(message, "rms.") {
			return true
		}
	}
	return false
}

func checkPayloadIsUsed(messages []string) bool {
	return true
}

func messageListToMessageDataList(messages []string) []MessageData {
	messagesData := make([]MessageData, len(messages))

	for pos, message := range messages {
		messageTypeSpliced := strings.SplitN(message, ".", 2)
		if len(messageTypeSpliced) != 2 {
			panic(throw.IllegalState())
		}

		messagesData[pos] = MessageData{
			TypeWithPackage: message,
			Type:            messageTypeSpliced[1],
		}
	}

	return messagesData
}

type MessageData struct {
	Type            string
	TypeWithPackage string
}

type GeneratorData struct {
	Messages      []MessageData
	RMSIsUsed     bool
	PayloadIsUsed bool
}

func generateMock(messages []string) (*bytes.Buffer, error) {
	data := GeneratorData{
		Messages:      messageListToMessageDataList(messages),
		RMSIsUsed:     checkRMSIsUsed(messages),
		PayloadIsUsed: checkPayloadIsUsed(messages),
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
