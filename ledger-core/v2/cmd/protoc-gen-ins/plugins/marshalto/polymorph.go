// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package marshalto

import (
	"strconv"

	"github.com/gogo/protobuf/protoc-gen-gogo/generator"

	"github.com/insolar/assured-ledger/ledger-core/v2/rms/insproto"
)

type polymorph struct {
	*generator.Generator
}

func (p *polymorph) Generate(message *generator.Descriptor, ccTypeName string) {
	if !insproto.HasPolymorphID(message.DescriptorProto) {
		return
	}
	id := insproto.GetPolymorphID(message.DescriptorProto)

	p.P(`func (m *`, ccTypeName, `) GetDefaultPolymorphID() uint64 {`)
	p.In()
	p.P(`return `, strconv.FormatUint(id, 10))
	p.Out()
	p.P(`}`)
}
