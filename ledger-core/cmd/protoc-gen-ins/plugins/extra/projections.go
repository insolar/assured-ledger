package extra

import (
	"fmt"
	"os"
	"strings"

	"github.com/gogo/protobuf/gogoproto"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"github.com/gogo/protobuf/vanity"

	"github.com/insolar/assured-ledger/ledger-core/insproto"
)

type Projection struct {
	*generator.Generator
}

func IsMessageProjection(file *descriptor.FileDescriptorProto, message *generator.Descriptor) bool {
	names := message.TypeName()
	return len(names) > 1 && insproto.IsProjection(file, message.DescriptorProto)
}

// This does Projection field mapping before any plugin.Generate()
// Mapping requires a map of all types, that is available on plugin.Init()

func (p *Projection) Init(g *generator.Generator) {
	files := p.Generator.Request.GetProtoFile()

	files = vanity.FilterFiles(files, vanity.NotGoogleProtobufDescriptorProto)
	vanity.ForEachFile(files, p.setFileProjections)
}

func (p *Projection) setFileProjections(file *descriptor.FileDescriptorProto) {
	for _, message := range file.GetMessageType() {
		for _, child := range message.GetNestedType() {
			p.setMessageProjections(file, message, child)
		}
	}
}

func (p *Projection) setMessageProjections(file *descriptor.FileDescriptorProto, parent, message *descriptor.DescriptorProto) {
	if insproto.IsProjection(file, message) {
		p.setMessageProjDesc(file, parent, message)
		// Projection can't have projections
		return
	}
	for _, child := range message.GetNestedType() {
		p.setMessageProjections(file, message, child)
	}
}

func (p *Projection) setMessageProjDesc(file *descriptor.FileDescriptorProto, parent, message *descriptor.DescriptorProto) {
	vanity.SetBoolMessageOption(gogoproto.E_Typedecl, false)(message)
	vanity.SetBoolMessageOption(gogoproto.E_GoprotoGetters, false)(message)
	vanity.SetBoolMessageOption(gogoproto.E_Face, true)(message)

	if insproto.GetPolymorphID(message) == 0 {
		if id := insproto.GetPolymorphID(parent); id > 0 {
			if err := proto.SetExtension(message.Options, insproto.E_Id, &id); err != nil {
				panic(err)
			}
		}
	}

	{
		context := insproto.GetCustomContext(file, parent)
		if err := proto.SetExtension(message.Options, insproto.E_Context, &context); err != nil {
			panic(err)
		}

		contextMethod := insproto.GetCustomContextMethod(file, parent)
		if err := proto.SetExtension(message.Options, insproto.E_ContextMethod, &contextMethod); err != nil {
			panic(err)
		}
	}

	fields := message.GetField()
	fieldMap := make(map[int32]*descriptor.FieldDescriptorProto, len(fields))

	for _, field := range fields {
		fieldMap[field.GetNumber()] = field
	}

	scanner := fieldScanner{gen: p.Generator, fieldMap: fieldMap}

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

func (p *Projection) Generate(file *generator.FileDescriptor, message *generator.Descriptor, ccTypeName string) {
	var projName []string
	var projections []string

	for _, subMsg := range message.GetNestedType() {
		if !insproto.IsProjection(file.FileDescriptorProto, subMsg) {
			continue
		}
		if len(projName) == 0 {
			projName = message.TypeName()
			projName = append(projName, "")
			p.P()
		}
		name := subMsg.GetName()
		projName[len(projName)-1] = name
		projections = append(projections, name)
		ccProjTypeName := generator.CamelCaseSlice(projName)

		p.P(`type `, ccTypeName, name, ` `, ccProjTypeName, `Face`)
		p.P(`type `, ccProjTypeName, ` `, ccTypeName)
		p.P()
		p.P(`func (m *`, ccTypeName, `) As`, name, `() *`, ccProjTypeName, ` {`)
		p.In()
		p.P(`return (*`, ccProjTypeName, `)(m)`)
		p.Out()
		p.P(`}`)
		p.P()
		p.P(`func (m *`, ccTypeName, `) As`, name, `Face() `, ccTypeName, name, ` {`)
		p.In()
		p.P(`if m == nil { return nil }`)
		p.P(`return (*`, ccProjTypeName, `)(m)`)
		p.Out()
		p.P(`}`)
		p.P()
		p.P(`func (m *`, ccProjTypeName, `) As`, message.GetName(), `() *`, ccTypeName, ` {`)
		p.In()
		p.P(`return (*`, ccTypeName, `)(m)`)
		p.Out()
		p.P(`}`)
		p.P()
		p.P(`func (m *`, ccProjTypeName, `) AsProjectionBase() interface{} {`)
		p.In()
		p.P(`if m == nil { return nil }`)
		p.P(`return (*`, ccTypeName, `)(m)`)
		p.Out()
		p.P(`}`)
		p.P()
	}

	if len(projections) > 0 {
		p.P()
		p.P(`func (m *`, ccTypeName, `) AsProjection(name string) interface{} {`)
		p.In()
		p.P(`if m == nil { return nil }`)
		p.P(`switch name {`)
		for _, name := range projections {
			p.P(`case "`, name, `":`)
			p.In()
			p.P(`return m.As`, name, `()`)
			p.Out()
		}
		p.P(`}`)
		p.P(`return nil`)
		p.Out()
		p.P(`}`)
		p.P()
	}
}

/************************************/

type fieldScannerEntry struct {
	msg    *descriptor.DescriptorProto
	prefix []string
}

type fieldScanner struct {
	gen      *generator.Generator
	fieldMap map[int32]*descriptor.FieldDescriptorProto
	types    []fieldScannerEntry
}

func (p *fieldScanner) findType(prefix []string, name string) *descriptor.DescriptorProto {
	if name[0] != '.' {
		name = `.` + strings.Join(prefix, `.`) + `.` + name
	}

	desc := p.gen.ObjectNamed(name).(*generator.Descriptor)
	return desc.DescriptorProto
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
			if parentField.IsMessage() && gogoproto.IsEmbed(parentField) && !gogoproto.IsNullable(parentField) {
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
