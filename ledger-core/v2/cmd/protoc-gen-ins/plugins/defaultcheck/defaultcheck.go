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

	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"

	"github.com/insolar/assured-ledger/ledger-core/v2/rms/insproto"
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

func (p *plugin) Generate(file *generator.FileDescriptor) {
	// proto3 := gogoproto.IsProto3(file.FileDescriptorProto)

	for _, msg := range file.Messages() {
		getters := gogoproto.HasGoGetters(file.FileDescriptorProto, msg.DescriptorProto)
		face := gogoproto.IsFace(file.FileDescriptorProto, msg.DescriptorProto)
		notation := insproto.IsNotation(file.FileDescriptorProto, msg.DescriptorProto)
		for _, field := range msg.GetField() {
			if notation && field.GetNumber() < 16 {
				fmt.Fprintf(os.Stderr, "ERROR: field %v.%v violates notation, field number < 16", generator.CamelCase(*msg.Name), generator.CamelCase(*field.Name))
				os.Exit(1)
			}

			if len(field.GetDefaultValue()) > 0 {
				if !getters {
					fmt.Fprintf(os.Stderr, "ERROR: field %v.%v cannot have a default value and not have a getter method", generator.CamelCase(*msg.Name), generator.CamelCase(*field.Name))
					os.Exit(1)
				}
				if face {
					fmt.Fprintf(os.Stderr, "ERROR: field %v.%v cannot have a default value be in a face", generator.CamelCase(*msg.Name), generator.CamelCase(*field.Name))
					os.Exit(1)
				}
			}
			if gogoproto.IsNullable(field) {
				if notation && field.GetNumber() < 20 {
					fmt.Fprintf(os.Stderr, "ERROR: field %v.%v violates notation, field with number [16..19] can't be nullable", generator.CamelCase(*msg.Name), generator.CamelCase(*field.Name))
					os.Exit(1)
				}
				continue
			}
			if len(field.GetDefaultValue()) > 0 {
				fmt.Fprintf(os.Stderr, "ERROR: field %v.%v cannot be non-nullable and have a default value", generator.CamelCase(*msg.Name), generator.CamelCase(*field.Name))
				os.Exit(1)
			}
			if !field.IsMessage() && !gogoproto.IsCustomType(field) {
				if field.IsRepeated() {
					fmt.Fprintf(os.Stderr, "WARNING: field %v.%v is a repeated non-nullable native type, nullable=false has no effect\n", generator.CamelCase(*msg.Name), generator.CamelCase(*field.Name))
					// } else if proto3 {
					// 	fmt.Fprintf(os.Stderr, "ERROR: field %v.%v is a native type and in proto3 syntax with nullable=false there exists conflicting implementations when encoding zero values", generator.CamelCase(*msg.Name), generator.CamelCase(*field.Name))
					// 	os.Exit(1)
				}
				if field.IsBytes() {
					fmt.Fprintf(os.Stderr, "WARNING: field %v.%v is a non-nullable bytes type, nullable=false has no effect\n", generator.CamelCase(*msg.Name), generator.CamelCase(*field.Name))
				}
			}
			if !field.IsEnum() {
				continue
			}
			enum := p.ObjectNamed(field.GetTypeName()).(*generator.EnumDescriptor)
			if len(enum.Value) == 0 || enum.Value[0].GetNumber() != 0 {
				fmt.Fprintf(os.Stderr, "ERROR: field %v.%v cannot be non-nullable and be an enum type %v which does not start with zero", generator.CamelCase(*msg.Name), generator.CamelCase(*field.Name), enum.GetName())
				os.Exit(1)
			}
		}
	}
	for _, e := range file.GetExtension() {
		if !gogoproto.IsNullable(e) {
			fmt.Fprintf(os.Stderr, "ERROR: extended field %v cannot be nullable %v", generator.CamelCase(e.GetName()), generator.CamelCase(*e.Name))
			os.Exit(1)
		}
	}
}

func (p *plugin) GenerateImports(*generator.FileDescriptor) {}
