// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package extra

import (
	"sort"
	"strconv"

	"github.com/gogo/protobuf/protoc-gen-gogo/generator"

	"github.com/insolar/assured-ledger/ledger-core/v2/insproto"
)

type Context struct {
	*generator.Generator
	generator.PluginImports
}

func (p *Context) Init(g *generator.Generator, imports generator.PluginImports) {
	p.Generator = g
	p.PluginImports = imports
}

func (p *Context) Generate(file *generator.FileDescriptor, message *generator.Descriptor, ccTypeName string) {
	customContext := insproto.GetCustomContext(file.FileDescriptorProto, message.DescriptorProto)
	if customContext == "" {
		return
	}
	customContextMethod := insproto.GetCustomContextMethod(file.FileDescriptorProto, message.DescriptorProto)
	if customContextMethod == "" {
		return
	}

	customContext = importCustomName(customContext, p.PluginImports)

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
		n := uint64(field.GetNumber())
		p.P(`if err := ctx.`, applyName, `(m, `, strconv.FormatUint(n, 10), `, &m.`, fieldName, `); err != nil {`)
		p.In()
		p.P(`return err`)
		p.Out()
		p.P(`}`)
	}

	id := insproto.GetPolymorphID(message.DescriptorProto)

	applyName := insproto.GetCustomMessageContextApply(file.FileDescriptorProto, message.DescriptorProto)
	if len(applyName) > 0 {
		p.P(`return ctx.`, applyName, `(m, `, strconv.FormatUint(id, 10), `)`)
	} else {
		p.P(`return nil`)
	}

	p.Out()
	p.P(`}`)
	p.P()
}
