// Protocol Buffers for Go with Gadgets
//
// Copyright (c) 2013, The GoGo Authors. All rights reserved.
// http://github.com/gogo/protobuf
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// This version is a modified by Insolar Network Ltd.

package defaultcheck

import (
	"fmt"
	"os"
	"sort"

	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"

	"github.com/insolar/assured-ledger/ledger-core/v2/cmd/protoc-gen-ins/plugins/extra"
	"github.com/insolar/assured-ledger/ledger-core/v2/insproto"
)

type plugin struct {
	*generator.Generator
}

func NewPlugin() *plugin {
	return &plugin{}
}

func (p *plugin) Name() string {
	return "defaultcheck"
}

func (p *plugin) Init(g *generator.Generator) {
	p.Generator = g
}

func printerr(m string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, m, args...)
}

func (p *plugin) Generate(file *generator.FileDescriptor) {
	proto3 := gogoproto.IsProto3(file.FileDescriptorProto)

	for _, message := range file.Messages() {
		if gogoproto.IsSizer(file.FileDescriptorProto, message.DescriptorProto) && gogoproto.IsProtoSizer(file.FileDescriptorProto, message.DescriptorProto) {
			printerr("ERROR: message %v cannot support both sizer and protosizer plugins\n", generator.CamelCase(*message.Name))
			os.Exit(1)
		}

		getters := gogoproto.HasGoGetters(file.FileDescriptorProto, message.DescriptorProto)
		face := gogoproto.IsFace(file.FileDescriptorProto, message.DescriptorProto)
		notation := insproto.IsNotation(file.FileDescriptorProto, message.DescriptorProto)
		has1619 := false
		hasFieldMap := false
		needsFieldMap := insproto.IsMappingForMessage(file.FileDescriptorProto, message.DescriptorProto)

		fields := message.GetField()
		if notation {
			sort.Sort(extra.OrderedFields(fields))
		}

		for _, field := range fields {
			if len(field.GetDefaultValue()) > 0 {
				if !getters {
					printerr("ERROR: field %v.%v cannot have a default value and not have a getter method", generator.CamelCase(*message.Name), generator.CamelCase(*field.Name))
					os.Exit(1)
				}
				if face {
					printerr("ERROR: field %v.%v cannot have a default value be in a face", generator.CamelCase(*message.Name), generator.CamelCase(*field.Name))
					os.Exit(1)
				}
			}

			if insproto.IsCustomContextApply(field) && !insproto.IsCustomContext(file.FileDescriptorProto, message.DescriptorProto) {
				printerr("WARNING: field %v.%v has delegate without context\n", generator.CamelCase(*message.Name), generator.CamelCase(*field.Name))
			}

			if field.GetTypeName() == insproto.FieldMapFQN {
				if field.GetName() != insproto.FieldMapFieldName || gogoproto.IsNullable(field) {
					printerr("ERROR: field %v.%v is illegal use of FieldMap\n", generator.CamelCase(*message.Name), generator.CamelCase(*field.Name))
					os.Exit(1)
				}
				hasFieldMap = true
			}

			needsFieldMap = needsFieldMap || insproto.IsMappingForField(field, message.DescriptorProto, file.FileDescriptorProto)

			if notation {
				switch fieldNumber := field.GetNumber(); {
				case fieldNumber < 16:
					printerr("ERROR: field %v.%v violates notation, field number < 16", generator.CamelCase(*message.Name), generator.CamelCase(*field.Name))
					os.Exit(1)
				case fieldNumber > 20:
					//
				case !has1619:
					if gogoproto.IsNullable(field) || field.IsRepeated() || field.IsPacked() || (proto3 && field.IsPacked3()) || field.DefaultValue != nil || field.OneofIndex != nil {
						printerr("ERROR: field %v.%v violates notation, first field with number [16..19] can't have be nullable, repeated, packed, oneof or with default value", generator.CamelCase(*message.Name), generator.CamelCase(*field.Name))
						os.Exit(1)
					}
					has1619 = true
				}
				// TODO check that incorporated messages also follow the notation?
			}

			if gogoproto.IsNullable(field) {
				continue
			}
			if len(field.GetDefaultValue()) > 0 {
				printerr("ERROR: field %v.%v cannot be non-nullable and have a default value", generator.CamelCase(*message.Name), generator.CamelCase(*field.Name))
				os.Exit(1)
			}
			if !field.IsEnum() {
				continue
			}
			enum := p.ObjectNamed(field.GetTypeName()).(*generator.EnumDescriptor)
			if len(enum.Value) == 0 || enum.Value[0].GetNumber() != 0 {
				printerr("ERROR: field %v.%v cannot be non-nullable and be an enum type %v which does not start with zero", generator.CamelCase(*message.Name), generator.CamelCase(*field.Name), enum.GetName())
				os.Exit(1)
			}
		}
		if notation && !has1619 && !insproto.HasPolymorphID(message.DescriptorProto) {
			printerr("ERROR: message %v violates notation, missing field with number [16..19]", generator.CamelCase(*message.Name))
			os.Exit(1)
		}
		if needsFieldMap && !hasFieldMap {
			printerr("ERROR: message %v field mapping requires FieldMap field\n", generator.CamelCase(*message.Name))
			os.Exit(1)
		}
	}
	for _, e := range file.GetExtension() {
		if !gogoproto.IsNullable(e) {
			printerr("ERROR: extended field %v cannot be nullable %v", generator.CamelCase(e.GetName()), generator.CamelCase(*e.Name))
			os.Exit(1)
		}
	}
}

func (p *plugin) GenerateImports(*generator.FileDescriptor) {}
