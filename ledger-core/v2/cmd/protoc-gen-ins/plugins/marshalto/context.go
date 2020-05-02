// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package marshalto

import (
	"sort"
	"strings"
	"unicode"

	"github.com/gogo/protobuf/protoc-gen-gogo/generator"

	"github.com/insolar/assured-ledger/ledger-core/v2/insproto"
)

type context struct {
	*generator.Generator
	generator.PluginImports
}

func (p *context) Generate(file *generator.FileDescriptor, message *generator.Descriptor, ccTypeName string) {
	customContext := insproto.GetCustomContext(file.FileDescriptorProto, message.DescriptorProto)
	if customContext == "" {
		return
	}
	customContextMethod := insproto.GetCustomContextMethod(file.FileDescriptorProto, message.DescriptorProto)
	if customContextMethod == "" {
		return
	}

	customContext = ImportCustomName(customContext, p.PluginImports)

	p.P(`func (m *`, ccTypeName, `) `, customContextMethod, `(ctx `, customContext, `) error {`)
	p.In()

	fields := OrderedFields(message.GetField())
	sort.Sort(fields)

	for _, field := range fields {
		applyName := insproto.GetCustomContextApply(field)
		if len(applyName) == 0 {
			continue
		}
		fieldName := p.GetFieldName(message, field)
		p.P(`if err := ctx.`, applyName, `(&m.`, fieldName, `); err != nil {`)
		p.In()
		p.P(`return err`)
		p.Out()
		p.P(`}`)
	}

	applyName := insproto.GetCustomMessageContextApply(file.FileDescriptorProto, message.DescriptorProto)
	if len(applyName) > 0 {
		p.P(`return ctx.`, applyName, `(m)`)
	} else {
		p.P(`return nil`)
	}

	p.Out()
	p.P(`}`)
	p.P()
}

func ImportCustomName(customName string, imports generator.PluginImports) string {
	packageName, typ := splitCPackageType(customName)
	if packageName != "" {
		pkg := imports.NewImport(packageName)
		pkg.Use()
		return typ
	}
	return typ
}

func splitCPackageType(ctype string) (packageName string, typ string) {
	lastDot := strings.LastIndexByte(ctype, '.')
	if lastDot < 0 {
		return "", ctype
	}
	packageName = ctype[:lastDot]
	importStr := strings.Map(badToUnderscore, packageName)
	typ = importStr + ctype[lastDot:]
	return packageName, typ
}

func badToUnderscore(r rune) rune {
	if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
		return r
	}
	return '_'
}
