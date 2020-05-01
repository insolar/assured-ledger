// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package marshalto

import (
	"sort"

	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"

	"github.com/insolar/assured-ledger/ledger-core/v2/rms/insproto"
)

type context struct {
	*generator.Generator
	generator.PluginImports
}

func (p *context) Generate(file *generator.FileDescriptor, message *generator.Descriptor, ccTypeName string) {
	customContext := insproto.GetCustomContext(message.DescriptorProto)
	if customContext == "" {
		return
	}
	customContextMethod := insproto.GetCustomContextMethod(message.DescriptorProto)
	if customContextMethod == "" {
		return
	}

	fakeField := &descriptor.FieldDescriptorProto{}
	fakeField.Options = &descriptor.FieldOptions{}
	if err := proto.SetExtension(fakeField.Options, gogoproto.E_Customtype, &customContext); err != nil {
		panic(err)
	}
	switch packageName, typ, err := generator.GetCustomType(fakeField); {
	case err != nil:
		panic(err)
	case packageName != "":
		p.NewImport(packageName).Use()
		fallthrough
	default:
		customContext = typ
	}

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

	p.P(`return nil`)
	p.Out()
	p.P(`}`)
}
