// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package extra

import (
	"strconv"

	"github.com/gogo/protobuf/protoc-gen-gogo/generator"

	"github.com/insolar/assured-ledger/ledger-core/insproto"
)

type Polymorph struct {
	*generator.Generator
	generator.PluginImports
	customRegistrations []string
}

func (p *Polymorph) Init(g *generator.Generator, imports generator.PluginImports) {
	p.Generator = g
	p.PluginImports = imports
	p.customRegistrations = nil
}

func (p *Polymorph) GenerateMsg(file *generator.FileDescriptor, message *generator.Descriptor, ccTypeName string, isHead bool) {
	id := insproto.GetPolymorphID(message.DescriptorProto)
	if id == 0 {
		return
	}
	idStr := strconv.FormatUint(id, 10)

	p.P(`const Type`, ccTypeName, `PolymorphID = `, idStr)
	p.P()

	p.P(`func (*`, ccTypeName, `) GetDefaultPolymorphID() uint64 {`)
	p.In()
	p.P(`return `, idStr)
	p.Out()
	p.P(`}`)
	p.P()

	if customReg := insproto.GetCustomRegister(file.FileDescriptorProto, message.DescriptorProto); customReg != "" {
		special := ``
		if isHead {
			special = message.GetName()
		}
		customReg = importCustomName(customReg, p.PluginImports)
		p.customRegistrations = append(p.customRegistrations,
			customReg+`(`+idStr+`, "`+special+`", (*`+ccTypeName+`)(nil))`)
	}
}

func (p *Polymorph) GenerateFile() {
	if len(p.customRegistrations) > 0 {
		p.P(`func init() {`)
		p.In()
		for _, s := range p.customRegistrations {
			p.P(s)
		}
		p.Out()
		p.P(`}`)
		p.P()
	}
}
