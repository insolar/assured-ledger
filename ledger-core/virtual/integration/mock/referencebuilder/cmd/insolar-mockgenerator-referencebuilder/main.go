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
	ProjectPath      = "github.com/insolar/assured-ledger/ledger-core"
	RecordsProtoFile = "rms/proto_records.pb.go"
	MockPath         = "virtual/integration/mock/referencebuilder/checker/typed.go"
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

func parseHandlers() ([]string, error) {
	projectPath, err := GetRealProjectDir()
	if err != nil {
		return []string{}, err
	}
	handlersPath := path.Join(projectPath, RecordsProtoFile)

	set := token.NewFileSet()
	fileInfo, err := parser.ParseFile(set, handlersPath, nil, 0)
	if err != nil {
		return []string{}, throw.W(err, "failed to parse file")
	}

	var records []string
	for name, object := range fileInfo.Scope.Objects {
		if name != "RecordExample" && name != "MemoryRecap" && object.Kind == ast.Typ {
			decl, ok := object.Decl.(*ast.TypeSpec)
			if !ok {
				panic(throw.IllegalState())
			}
			_, isStruct := decl.Type.(*ast.StructType)
			if isStruct {
				records = append(records, name)
			}
		}
	}

	sort.Slice(records, func(i, j int) bool { return strings.Compare(records[i], records[j]) < 0 })

	return records, nil
}

type GeneratorData struct {
	Messages []string
}

func generateMock(messages []string) (*bytes.Buffer, error) {
	data := GeneratorData{
		Messages: messages,
	}

	var (
		tpl = template.New("typed.go.tpl")
		buf = bytes.Buffer{}
	)

	_, err := tpl.Parse(ReferenceBuilderMock)
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
	handlers, err := parseHandlers()
	if err != nil {
		panic(err)
	}

	mock, err := generateMock(handlers)
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
