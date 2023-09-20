// This version was modified by Insolar Network Ltd.

package defaultcheck

import (
	"fmt"
	"os"
	"sort"

	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"

	"github.com/insolar/assured-ledger/ledger-core/cmd/protoc-gen-ins/plugins/extra"
	"github.com/insolar/assured-ledger/ledger-core/insproto"
)

type plugin struct {
	*generator.Generator
}

func NewPlugin() generator.Plugin {
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
		projection := insproto.IsProjection(file.FileDescriptorProto, message.DescriptorProto)
		hasFieldMap := false
		needsFieldMap := !projection && insproto.IsMappingForMessage(file.FileDescriptorProto, message.DescriptorProto)

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
				printerr("WARNING: field %v.%v has context apply without a context type\n", generator.CamelCase(*message.Name), generator.CamelCase(*field.Name))
			}

			if field.GetTypeName() == insproto.FieldMapFQN {
				if field.GetName() != insproto.FieldMapFieldName || !gogoproto.IsNullable(field) {
					printerr("ERROR: field %v.%v is illegal use of FieldMap\n", generator.CamelCase(*message.Name), generator.CamelCase(*field.Name))
					os.Exit(1)
				}
				hasFieldMap = true
			}

			if !projection {
				needsFieldMap = needsFieldMap || insproto.IsMappingForField(field, message.DescriptorProto, file.FileDescriptorProto)
			}

			if notation {
				switch fieldNumber := field.GetNumber(); {
				case fieldNumber < 16:
					printerr("ERROR: field %v.%v violates notation, field number < 16", generator.CamelCase(*message.Name), generator.CamelCase(*field.Name))
					os.Exit(1)
				case fieldNumber > 20:
					//
				case !has1619:
					if gogoproto.IsNullable(field) || field.IsRepeated() || field.IsPacked() || (proto3 && field.IsPacked3()) || field.DefaultValue != nil || field.OneofIndex != nil {
						printerr("ERROR: field %v.%v violates notation, first field with number [16..19] can't be nullable, repeated, packed, oneof or with default value", generator.CamelCase(*message.Name), generator.CamelCase(*field.Name))
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
				printerr("ERROR: field %v.%v can't be non-nullable and have a default value", generator.CamelCase(*message.Name), generator.CamelCase(*field.Name))
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
