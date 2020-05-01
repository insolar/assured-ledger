// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package marshalto

import (
	"fmt"
	"os"
	"strings"

	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"github.com/gogo/protobuf/vanity"

	"github.com/insolar/assured-ledger/ledger-core/v2/insproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type head struct {
	*generator.Generator
}

func IsMessageHead(file *generator.FileDescriptor, message *generator.Descriptor) bool {
	names := message.TypeName()
	return len(names) > 1 && insproto.IsHead(message.DescriptorProto)
}

func SetMessageHeadDesc(file *descriptor.FileDescriptorProto, parent, message *descriptor.DescriptorProto) {
	vanity.SetBoolMessageOption(gogoproto.E_Typedecl, false)(message)
	vanity.SetBoolMessageOption(gogoproto.E_Face, true)(message)

	if insproto.GetPolymorphID(message) == 0 {
		if id := insproto.GetPolymorphID(parent); id > 0 {
			if err := proto.SetExtension(message.Options, insproto.E_Id, &id); err != nil {
				panic(err)
			}
		}
	}

	fields := message.GetField()
	fieldMap := make(map[int32]*descriptor.FieldDescriptorProto, len(fields))

	for _, field := range fields {
		fieldMap[field.GetNumber()] = field
	}

	scanner := fieldScanner{file: file, fieldMap: fieldMap}

	if err := scanner.findAll(parent); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s.%s", message.GetName(), err.Error())
		os.Exit(1)
	}

	if len(fieldMap) != 0 {
		missingFields := strings.Builder{}
		for _, field := range fieldMap {
			missingFields.WriteByte(' ')
			missingFields.WriteString(field.GetName())
		}
		_, _ = fmt.Fprintf(os.Stderr, "%s of parent %s has missing fields%s", message.GetName(), parent.GetName(), missingFields.String())
		os.Exit(1)
	}
}

func (p *head) Generate(message *generator.Descriptor, ccTypeName string) {
	var headName []string

	for _, subMsg := range message.GetNestedType() {
		if !insproto.IsHead(subMsg) {
			continue
		}
		if len(headName) == 0 {
			headName = message.TypeName()
			headName = append(headName, "")
			p.P()
		}
		name := subMsg.GetName()
		headName[len(headName)-1] = name
		ccHeadTypeName := generator.CamelCaseSlice(headName)

		p.P(`type `, ccTypeName, name, ` = `, ccHeadTypeName, `Face`)
		p.P()
		p.P(`type `, ccHeadTypeName, ` struct {`)
		p.In()
		p.P(ccTypeName)
		p.Out()
		p.P(`}`)
		p.P()
	}

	if len(headName) > 0 {
		p.P()
	}
}

type fieldScannerEntry struct {
	msg    *descriptor.DescriptorProto
	prefix []string
}

type fieldScanner struct {
	file     *descriptor.FileDescriptorProto
	fieldMap map[int32]*descriptor.FieldDescriptorProto
	types    []fieldScannerEntry
}

func (p *fieldScanner) findType(prefix []string, name string) *descriptor.DescriptorProto {
	if name[0] == '.' {
		if start := strings.IndexByte(name[1:], '.'); start > 0 {
			msg := p.file.GetMessage(name[start+2:])
			if msg != nil {
				return msg
			}
		}
		panic(throw.Impossible())
	}

	for i := len(prefix); i >= 0; i-- {
		typeName := name
		if i > 0 {
			typeName = strings.Join(prefix[:i], `.`) + `.` + name
		}
		msg := p.file.GetMessage(typeName)
		if msg != nil {
			return msg
		}
	}
	panic(throw.Impossible())
}

func (p *fieldScanner) popField(n int32) *descriptor.FieldDescriptorProto {
	field := p.fieldMap[n]
	if field == nil {
		return nil
	}
	delete(p.fieldMap, n)
	return field
}

func (p *fieldScanner) hasFields() bool {
	return len(p.fieldMap) > 0
}

func (p *fieldScanner) scanFields(prefix []string, parent *descriptor.DescriptorProto) error {
	for _, parentField := range parent.GetField() {
		field := p.popField(parentField.GetNumber())
		if field == nil {
			if parentField.IsMessage() && gogoproto.IsEmbed(parentField) {
				typeName := parentField.GetTypeName()
				embedded := p.findType(prefix, typeName)
				if pos := strings.LastIndexByte(typeName, '.'); pos >= 0 {
					typeName = typeName[pos+1:]
				}
				prefix = append(prefix, typeName)
				p.types = append(p.types, fieldScannerEntry{embedded, prefix})
			}
			continue
		}
		var err string
		switch {
		case field.OneofIndex != nil:
			// parent oneof is irrelevant
			err = "oneof is not supported"
		case field.GetName() != parentField.GetName():
			err = "name mismatch"
		case field.GetType() != parentField.GetType():
			err = "type mismatch"
		case field.Label != nil && field.GetLabel() != parentField.GetLabel():
			err = "label is different"
		case field.DefaultValue != nil && field.GetDefaultValue() != parentField.GetDefaultValue():
			err = "default value is different"
		case field.JsonName != nil && field.GetJsonName() != parentField.GetJsonName():
			err = "json name is different"
		case field.Extendee != nil && field.GetExtendee() != parentField.GetExtendee():
			err = "extendee is different"
		case field.Options != nil:
			err = "options are not allowed"
		default:
			*field = *parentField
			if !p.hasFields() {
				return nil
			}
			continue
		}
		return fmt.Errorf("%s is incompatible with parent %s.%s, %s", field.GetName(),
			parent.GetName(), parentField.GetName(), err)
	}
	return nil
}

func (p *fieldScanner) findAll(parent *descriptor.DescriptorProto) error {
	p.types = []fieldScannerEntry{{msg: parent}}
	for i := 0; i < len(p.types) && p.hasFields(); i++ {
		t := p.types[i]
		if err := p.scanFields(t.prefix, t.msg); err != nil {
			return err
		}
	}
	return nil
}
