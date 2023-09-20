// This version is a modified by Insolar Network Ltd.

/*
The marshalto plugin generates a Marshal and MarshalTo method for each message.
The `Marshal() ([]byte, error)` method results in the fact that the message
implements the Marshaler interface.
This allows proto.Marshal to be faster by calling the generated Marshal method rather than using reflect to Marshal the struct.

If is enabled by the following extensions:

  - marshaler
  - marshaler_all

Or the following extensions:

  - unsafe_marshaler
  - unsafe_marshaler_all

That is if you want to use the unsafe package in your generated code.
The speed up using the unsafe package is not very significant.

*/

// This version was modified by Insolar Network Ltd.

//nolint // the most of code is reused from gogo-proto
package gogobased

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/gogoproto"
	marshal "github.com/gogo/protobuf/plugin/marshalto"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"
	"github.com/gogo/protobuf/protoc-gen-gogo/generator"
	"github.com/gogo/protobuf/vanity"

	"github.com/insolar/assured-ledger/ledger-core/cmd/protoc-gen-ins/plugins/extra"
	"github.com/insolar/assured-ledger/ledger-core/insproto"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type marshalto struct {
	*generator.Generator
	generator.PluginImports

	projections extra.Projection
	polyGen     extra.Polymorph
	contextGen  extra.Context

	atleastOne  bool
	errorsPkg   generator.Single
	protoPkg    generator.Single
	sortKeysPkg generator.Single
	mathPkg     generator.Single
	typesPkg    generator.Single
	binaryPkg   generator.Single
	fieldMapPkg generator.Single
	localName   string
}

// TODO support "raw_bytes"

func NewMarshal() *marshalto {
	return &marshalto{}
}

func (p *marshalto) Name() string {
	return "marshalto"
}

func (p *marshalto) Init(g *generator.Generator) {
	p.Generator = g
	p.projections.Generator = g
	p.projections.Init(g) // runs Projection propagation
}

func (p *marshalto) callFixed64(varName ...string) {
	p.P(`i -= 8`)
	p.P(p.binaryPkg.Use(), `.LittleEndian.PutUint64(dAtA[i:], uint64(`, strings.Join(varName, ""), `))`)
}

func (p *marshalto) callFixed32(varName ...string) {
	p.P(`i -= 4`)
	p.P(p.binaryPkg.Use(), `.LittleEndian.PutUint32(dAtA[i:], uint32(`, strings.Join(varName, ""), `))`)
}

func (p *marshalto) callVarint(varName ...string) {
	p.P(`i = encodeVarint`, p.localName, `(dAtA, i, uint64(`, strings.Join(varName, ""), `))`)
}

func (p *marshalto) encodeKey(fieldNumber int32, wireType int) {
	x := uint32(fieldNumber)<<3 | uint32(wireType)
	i := 0
	keybuf := make([]byte, 0)
	for i = 0; x > 127; i++ {
		keybuf = append(keybuf, 0x80|uint8(x&0x7F))
		x >>= 7
	}
	keybuf = append(keybuf, uint8(x))
	for i = len(keybuf) - 1; i >= 0; i-- {
		p.P(`i--`)
		p.P(`dAtA[i] = `, fmt.Sprintf("%#v", keybuf[i]))
	}
}

func (p *marshalto) mapField(numGen marshal.NumGen, field *descriptor.FieldDescriptorProto, kvField *descriptor.FieldDescriptorProto, varName string, protoSizer bool) {
	switch kvField.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		p.callFixed64(p.mathPkg.Use(), `.Float64bits(float64(`, varName, `))`)
	case descriptor.FieldDescriptorProto_TYPE_FLOAT:
		p.callFixed32(p.mathPkg.Use(), `.Float32bits(float32(`, varName, `))`)
	case descriptor.FieldDescriptorProto_TYPE_INT64,
		descriptor.FieldDescriptorProto_TYPE_UINT64,
		descriptor.FieldDescriptorProto_TYPE_INT32,
		descriptor.FieldDescriptorProto_TYPE_UINT32,
		descriptor.FieldDescriptorProto_TYPE_ENUM:
		p.callVarint(varName)
	case descriptor.FieldDescriptorProto_TYPE_FIXED64,
		descriptor.FieldDescriptorProto_TYPE_SFIXED64:
		p.callFixed64(varName)
	case descriptor.FieldDescriptorProto_TYPE_FIXED32,
		descriptor.FieldDescriptorProto_TYPE_SFIXED32:
		p.callFixed32(varName)
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		p.P(`i--`)
		p.P(`if `, varName, ` {`)
		p.In()
		p.P(`dAtA[i] = 1`)
		p.Out()
		p.P(`} else {`)
		p.In()
		p.P(`dAtA[i] = 0`)
		p.Out()
		p.P(`}`)
	case descriptor.FieldDescriptorProto_TYPE_STRING,
		descriptor.FieldDescriptorProto_TYPE_BYTES:
		if gogoproto.IsCustomType(field) && kvField.IsBytes() {
			p.forward(varName, true, protoSizer)
		} else {
			p.P(`i -= len(`, varName, `)`)
			p.P(`copy(dAtA[i:], `, varName, `)`)
			p.callVarint(`len(`, varName, `)`)
		}
	case descriptor.FieldDescriptorProto_TYPE_SINT32:
		p.callVarint(`(uint32(`, varName, `) << 1) ^ uint32((`, varName, ` >> 31))`)
	case descriptor.FieldDescriptorProto_TYPE_SINT64:
		p.callVarint(`(uint64(`, varName, `) << 1) ^ uint64((`, varName, ` >> 63))`)
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		if !p.marshalAllSizeOf(kvField, `(*`+varName+`)`, numGen.Next()) {
			if gogoproto.IsCustomType(field) {
				p.forward(varName, true, protoSizer)
			} else {
				p.backward(varName, true)
			}
		}

	}
}

func (p *marshalto) generateField(proto3, notation, zeroableDefault, protoSizer, hasFieldMap, mustBeIncluded bool,
	numGen marshal.NumGen, file *generator.FileDescriptor, message *generator.Descriptor, field *descriptor.FieldDescriptorProto,
) {
	fieldname := p.GetOneOfFieldName(message, field)
	nullable := gogoproto.IsNullable(field)
	repeated := field.IsRepeated()
	required := field.IsRequired()

	needsMapping := hasFieldMap && insproto.IsMappingForField(field, message.DescriptorProto, file.FileDescriptorProto)
	if needsMapping {
		p.P(`fieldEnd = i`)
	}

	doNilCheck := gogoproto.NeedsNilCheck(proto3, field)
	if required && nullable {
		p.P(`if m.`, fieldname, `== nil {`)
		p.In()
		if !gogoproto.ImportsGoGoProto(file.FileDescriptorProto) {
			p.P(`return 0, new(`, p.protoPkg.Use(), `.RequiredNotSetError)`)
		} else {
			p.P(`return 0, `, p.protoPkg.Use(), `.NewRequiredNotSetError("`, field.GetName(), `")`)
		}
		p.Out()
		p.P(`} else {`)
	} else if repeated {
		p.P(`if len(m.`, fieldname, `) > 0 {`)
		p.In()
	} else if doNilCheck {
		p.P(`if m.`, fieldname, ` != nil {`)
		p.In()
	}
	packed := field.IsPacked() || (proto3 && field.IsPacked3())
	wireType := field.WireType()
	fieldNumber := field.GetNumber()
	if packed {
		wireType = proto.WireBytes
	}

	switch *field.Type {
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		if packed {
			val := p.reverseListRange(`m.`, fieldname)
			p.P(`f`, numGen.Next(), ` := `, p.mathPkg.Use(), `.Float64bits(float64(`, val, `))`)
			p.callFixed64("f" + numGen.Current())
			p.Out()
			p.P(`}`)
			p.callVarint(`len(m.`, fieldname, `) * 8`)
			p.encodeKey(fieldNumber, wireType)
		} else if repeated {
			val := p.reverseListRange(`m.`, fieldname)
			p.P(`f`, numGen.Next(), ` := `, p.mathPkg.Use(), `.Float64bits(float64(`, val, `))`)
			p.callFixed64("f" + numGen.Current())
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else if proto3 && !mustBeIncluded {
			p.P(`if m.`, fieldname, ` != 0 {`)
			p.In()
			p.callFixed64(p.mathPkg.Use(), `.Float64bits(float64(m.`+fieldname, `))`)
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else if !nullable {
			p.callFixed64(p.mathPkg.Use(), `.Float64bits(float64(m.`+fieldname, `))`)
			p.encodeKey(fieldNumber, wireType)
		} else {
			p.callFixed64(p.mathPkg.Use(), `.Float64bits(float64(*m.`+fieldname, `))`)
			p.encodeKey(fieldNumber, wireType)
		}
	case descriptor.FieldDescriptorProto_TYPE_FLOAT:
		if packed {
			val := p.reverseListRange(`m.`, fieldname)
			p.P(`f`, numGen.Next(), ` := `, p.mathPkg.Use(), `.Float32bits(float32(`, val, `))`)
			p.callFixed32("f" + numGen.Current())
			p.Out()
			p.P(`}`)
			p.callVarint(`len(m.`, fieldname, `) * 4`)
			p.encodeKey(fieldNumber, wireType)
		} else if repeated {
			val := p.reverseListRange(`m.`, fieldname)
			p.P(`f`, numGen.Next(), ` := `, p.mathPkg.Use(), `.Float32bits(float32(`, val, `))`)
			p.callFixed32("f" + numGen.Current())
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else if proto3 && !mustBeIncluded {
			p.P(`if m.`, fieldname, ` != 0 {`)
			p.In()
			p.callFixed32(p.mathPkg.Use(), `.Float32bits(float32(m.`+fieldname, `))`)
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else if !nullable {
			p.callFixed32(p.mathPkg.Use(), `.Float32bits(float32(m.`+fieldname, `))`)
			p.encodeKey(fieldNumber, wireType)
		} else {
			p.callFixed32(p.mathPkg.Use(), `.Float32bits(float32(*m.`+fieldname, `))`)
			p.encodeKey(fieldNumber, wireType)
		}
	case descriptor.FieldDescriptorProto_TYPE_INT64,
		descriptor.FieldDescriptorProto_TYPE_UINT64,
		descriptor.FieldDescriptorProto_TYPE_INT32,
		descriptor.FieldDescriptorProto_TYPE_UINT32,
		descriptor.FieldDescriptorProto_TYPE_ENUM:
		if packed {
			jvar := "j" + numGen.Next()
			p.P(`dAtA`, numGen.Next(), ` := make([]byte, len(m.`, fieldname, `)*10)`)
			p.P(`var `, jvar, ` int`)
			if *field.Type == descriptor.FieldDescriptorProto_TYPE_INT64 ||
				*field.Type == descriptor.FieldDescriptorProto_TYPE_INT32 {
				p.P(`for _, num1 := range m.`, fieldname, ` {`)
				p.In()
				p.P(`num := uint64(num1)`)
			} else {
				p.P(`for _, num := range m.`, fieldname, ` {`)
				p.In()
			}
			p.P(`for num >= 1<<7 {`)
			p.In()
			p.P(`dAtA`, numGen.Current(), `[`, jvar, `] = uint8(uint64(num)&0x7f|0x80)`)
			p.P(`num >>= 7`)
			p.P(jvar, `++`)
			p.Out()
			p.P(`}`)
			p.P(`dAtA`, numGen.Current(), `[`, jvar, `] = uint8(num)`)
			p.P(jvar, `++`)
			p.Out()
			p.P(`}`)
			p.P(`i -= `, jvar)
			p.P(`copy(dAtA[i:], dAtA`, numGen.Current(), `[:`, jvar, `])`)
			p.callVarint(jvar)
			p.encodeKey(fieldNumber, wireType)
		} else if repeated {
			val := p.reverseListRange(`m.`, fieldname)
			p.callVarint(val)
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else if proto3 && !mustBeIncluded {
			p.P(`if m.`, fieldname, ` != 0 {`)
			p.In()
			p.callVarint(`m.`, fieldname)
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else if !nullable {
			p.callVarint(`m.`, fieldname)
			p.encodeKey(fieldNumber, wireType)
		} else {
			p.callVarint(`*m.`, fieldname)
			p.encodeKey(fieldNumber, wireType)
		}
	case descriptor.FieldDescriptorProto_TYPE_FIXED64,
		descriptor.FieldDescriptorProto_TYPE_SFIXED64:
		if packed {
			val := p.reverseListRange(`m.`, fieldname)
			p.callFixed64(val)
			p.Out()
			p.P(`}`)
			p.callVarint(`len(m.`, fieldname, `) * 8`)
			p.encodeKey(fieldNumber, wireType)
		} else if repeated {
			val := p.reverseListRange(`m.`, fieldname)
			p.callFixed64(val)
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else if proto3 && !mustBeIncluded {
			p.P(`if m.`, fieldname, ` != 0 {`)
			p.In()
			p.callFixed64("m." + fieldname)
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else if !nullable {
			p.callFixed64("m." + fieldname)
			p.encodeKey(fieldNumber, wireType)
		} else {
			p.callFixed64("*m." + fieldname)
			p.encodeKey(fieldNumber, wireType)
		}
	case descriptor.FieldDescriptorProto_TYPE_FIXED32,
		descriptor.FieldDescriptorProto_TYPE_SFIXED32:
		if packed {
			val := p.reverseListRange(`m.`, fieldname)
			p.callFixed32(val)
			p.Out()
			p.P(`}`)
			p.callVarint(`len(m.`, fieldname, `) * 4`)
			p.encodeKey(fieldNumber, wireType)
		} else if repeated {
			val := p.reverseListRange(`m.`, fieldname)
			p.callFixed32(val)
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else if proto3 && !mustBeIncluded {
			p.P(`if m.`, fieldname, ` != 0 {`)
			p.In()
			p.callFixed32("m." + fieldname)
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else if !nullable {
			p.callFixed32("m." + fieldname)
			p.encodeKey(fieldNumber, wireType)
		} else {
			p.callFixed32("*m." + fieldname)
			p.encodeKey(fieldNumber, wireType)
		}
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		if packed {
			val := p.reverseListRange(`m.`, fieldname)
			p.P(`i--`)
			p.P(`if `, val, ` {`)
			p.In()
			p.P(`dAtA[i] = 1`)
			p.Out()
			p.P(`} else {`)
			p.In()
			p.P(`dAtA[i] = 0`)
			p.Out()
			p.P(`}`)
			p.Out()
			p.P(`}`)
			p.callVarint(`len(m.`, fieldname, `)`)
			p.encodeKey(fieldNumber, wireType)
		} else if repeated {
			val := p.reverseListRange(`m.`, fieldname)
			p.P(`i--`)
			p.P(`if `, val, ` {`)
			p.In()
			p.P(`dAtA[i] = 1`)
			p.Out()
			p.P(`} else {`)
			p.In()
			p.P(`dAtA[i] = 0`)
			p.Out()
			p.P(`}`)
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else if proto3 && !mustBeIncluded {
			p.P(`if m.`, fieldname, ` {`)
			p.In()
			p.P(`i--`)
			p.P(`if m.`, fieldname, ` {`)
			p.In()
			p.P(`dAtA[i] = 1`)
			p.Out()
			p.P(`} else {`)
			p.In()
			p.P(`dAtA[i] = 0`)
			p.Out()
			p.P(`}`)
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else if !nullable {
			p.P(`i--`)
			p.P(`if m.`, fieldname, ` {`)
			p.In()
			p.P(`dAtA[i] = 1`)
			p.Out()
			p.P(`} else {`)
			p.In()
			p.P(`dAtA[i] = 0`)
			p.Out()
			p.P(`}`)
			p.encodeKey(fieldNumber, wireType)
		} else {
			p.P(`i--`)
			p.P(`if *m.`, fieldname, ` {`)
			p.In()
			p.P(`dAtA[i] = 1`)
			p.Out()
			p.P(`} else {`)
			p.In()
			p.P(`dAtA[i] = 0`)
			p.Out()
			p.P(`}`)
			p.encodeKey(fieldNumber, wireType)
		}
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		isRaw := insproto.IsRawBytes(field, true)
		if repeated {
			val := p.reverseListRange(`m.`, fieldname)
			p.P(`i -= len(`, val, `)`)
			p.P(`copy(dAtA[i:], `, val, `)`)
			if isRaw {
				p.callVarint(`len(`, val, `)`)
			} else {
				p.P(`i--`)
				p.P(`dAtA[i]=`, int(protokit.BinaryMarker))
				p.callVarint(`len(`, val, `)+1`)
			}
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else {
			hasIf := proto3 && !mustBeIncluded
			prefix := `m.`
			if !hasIf && nullable {
				// this check is a bit strange, but it mimics gogo behavior
				prefix = `*m.`
			}
			if hasIf {
				p.P(`if len(`, prefix, fieldname, `) > 0 {`)
				p.In()
			}
			p.P(`i -= len(`, prefix, fieldname, `)`)
			p.P(`copy(dAtA[i:], `, prefix, fieldname, `)`)
			if isRaw {
				p.callVarint(`len(`, prefix, fieldname, `)`)
			} else {
				p.P(`i--`)
				p.P(`dAtA[i]=`, int(protokit.BinaryMarker))
				p.callVarint(`len(`, prefix, fieldname, `)+1`)
			}
			p.encodeKey(fieldNumber, wireType)
			if hasIf {
				p.Out()
				p.P(`}`)
			}
		}
	case descriptor.FieldDescriptorProto_TYPE_GROUP:
		panic(fmt.Errorf("marshaler does not support group %v", fieldname))
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		if p.IsMap(field) {
			m := p.GoMapType(nil, field)
			keygoTyp, keywire := p.GoType(nil, m.KeyField)
			keygoAliasTyp, _ := p.GoType(nil, m.KeyAliasField)
			// keys may not be pointers
			keygoTyp = strings.Replace(keygoTyp, "*", "", 1)
			keygoAliasTyp = strings.Replace(keygoAliasTyp, "*", "", 1)
			keyCapTyp := generator.CamelCase(keygoTyp)
			valuegoTyp, valuewire := p.GoType(nil, m.ValueField)
			valuegoAliasTyp, _ := p.GoType(nil, m.ValueAliasField)
			nullable, valuegoTyp, valuegoAliasTyp = generator.GoMapValueTypes(field, m.ValueField, valuegoTyp, valuegoAliasTyp)
			var val string
			if gogoproto.IsStableMarshaler(file.FileDescriptorProto, message.DescriptorProto) {
				keysName := `keysFor` + fieldname
				p.P(keysName, ` := make([]`, keygoTyp, `, 0, len(m.`, fieldname, `))`)
				p.P(`for k := range m.`, fieldname, ` {`)
				p.In()
				p.P(keysName, ` = append(`, keysName, `, `, keygoTyp, `(k))`)
				p.Out()
				p.P(`}`)
				p.P(p.sortKeysPkg.Use(), `.`, keyCapTyp, `s(`, keysName, `)`)
				val = p.reverseListRange(keysName)
			} else {
				p.P(`for k := range m.`, fieldname, ` {`)
				val = "k"
				p.In()
			}
			if gogoproto.IsStableMarshaler(file.FileDescriptorProto, message.DescriptorProto) {
				p.P(`v := m.`, fieldname, `[`, keygoAliasTyp, `(`, val, `)]`)
			} else {
				p.P(`v := m.`, fieldname, `[`, val, `]`)
			}
			p.P(`baseI := i`)
			accessor := `v`

			if m.ValueField.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE {
				if valuegoTyp != valuegoAliasTyp && !gogoproto.IsStdType(m.ValueAliasField) {
					if nullable {
						// cast back to the type that has the generated methods on it
						accessor = `((` + valuegoTyp + `)(` + accessor + `))`
					} else {
						accessor = `((*` + valuegoTyp + `)(&` + accessor + `))`
					}
				} else if !nullable {
					accessor = `(&v)`
				}
			}

			nullableMsg := nullable && (m.ValueField.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE ||
				gogoproto.IsCustomType(field) && m.ValueField.IsBytes())

			plainBytes := m.ValueField.IsBytes() && !gogoproto.IsCustomType(field)
			if nullableMsg {
				p.P(`if `, accessor, ` != nil { `)
				p.In()
			} else if plainBytes {
				if proto3 {
					p.P(`if len(`, accessor, `) > 0 {`)
				} else {
					p.P(`if `, accessor, ` != nil {`)
				}
				p.In()
			}
			p.mapField(numGen, field, m.ValueAliasField, accessor, protoSizer)
			p.encodeKey(2, wireToType(valuewire))
			if nullableMsg || plainBytes {
				p.Out()
				p.P(`}`)
			}

			p.mapField(numGen, field, m.KeyField, val, protoSizer)
			p.encodeKey(1, wireToType(keywire))

			p.callVarint(`baseI - i`)

			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else if repeated {
			val := p.reverseListRange(`m.`, fieldname)
			sizeOfVarName := val
			if gogoproto.IsNullable(field) {
				sizeOfVarName = `*` + val
			}
			if !p.marshalAllSizeOf(field, sizeOfVarName, ``) {
				if gogoproto.IsCustomType(field) {
					p.forward(val, true, protoSizer)
				} else {
					p.backward(val, true)
				}
			}
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else {
			sizeOfVarName := `m.` + fieldname
			if gogoproto.IsNullable(field) {
				sizeOfVarName = `*` + sizeOfVarName
			}
			if !p.marshalAllSizeOf(field, sizeOfVarName, numGen.Next()) {
				if !required && !mustBeIncluded && insproto.IsZeroable(field, zeroableDefault) {
					keyFn := func() {
						p.encodeKey(fieldNumber, wireType)
					}
					if gogoproto.IsCustomType(field) {
						p.forwardZeroable(`m.`+fieldname, true, protoSizer, true, keyFn)
					} else {
						p.backwardZeroable(`m.`+fieldname, true, keyFn)
					}
					break
				}
				if gogoproto.IsCustomType(field) {
					p.forward(`m.`+fieldname, true, protoSizer)
				} else {
					p.backward(`m.`+fieldname, true)
				}
			}
			p.encodeKey(fieldNumber, wireType)
		}
	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		if !gogoproto.IsCustomType(field) {
			isRaw := insproto.IsRawBytes(field, !notation)
			if repeated {
				val := p.reverseListRange(`m.`, fieldname)
				p.P(`i -= len(`, val, `)`)
				p.P(`copy(dAtA[i:], `, val, `)`)
				if isRaw {
					p.callVarint(`len(`, val, `)`)
				} else {
					p.P(`i--`)
					p.P(`dAtA[i]=`, int(protokit.BinaryMarker))
					p.callVarint(`len(`, val, `)+1`)
				}
				p.encodeKey(fieldNumber, wireType)
				p.Out()
				p.P(`}`)
			} else {
				hasIf := proto3 && !mustBeIncluded
				if hasIf {
					p.P(`if len(m.`, fieldname, `) > 0 {`)
					p.In()
				}
				p.P(`i -= len(m.`, fieldname, `)`)
				p.P(`copy(dAtA[i:], m.`, fieldname, `)`)
				if isRaw {
					p.callVarint(`len(m.`, fieldname, `)`)
				} else {
					p.P(`i--`)
					p.P(`dAtA[i]=`, int(protokit.BinaryMarker))
					p.callVarint(`len(m.`, fieldname, `)+1`)
				}
				p.encodeKey(fieldNumber, wireType)
				if hasIf {
					p.Out()
					p.P(`}`)
				}
			}
		} else {
			isRaw := insproto.IsRawBytes(field, true)
			if repeated {
				val := p.reverseListRange(`m.`, fieldname)
				p.forwardZeroable(val, true, protoSizer, isRaw, nil)
				p.encodeKey(fieldNumber, wireType)
				p.Out()
				p.P(`}`)
			} else {
				p.forwardZeroable(`m.`+fieldname, true, protoSizer, isRaw, nil)
				p.encodeKey(fieldNumber, wireType)
			}
		}
	case descriptor.FieldDescriptorProto_TYPE_SINT32:
		if packed {
			datavar := "dAtA" + numGen.Next()
			jvar := "j" + numGen.Next()
			p.P(datavar, ` := make([]byte, len(m.`, fieldname, ")*5)")
			p.P(`var `, jvar, ` int`)
			p.P(`for _, num := range m.`, fieldname, ` {`)
			p.In()
			xvar := "x" + numGen.Next()
			p.P(xvar, ` := (uint32(num) << 1) ^ (uint32(num) >> 31)`)
			p.P(`for `, xvar, ` >= 1<<7 {`)
			p.In()
			p.P(datavar, `[`, jvar, `] = uint8(`, xvar, `|0x80)`)
			p.P(jvar, `++`)
			p.P(xvar, ` >>= 7`)
			p.Out()
			p.P(`}`)
			p.P(datavar, `[`, jvar, `] = uint8(`, xvar, `)`)
			p.P(jvar, `++`)
			p.Out()
			p.P(`}`)
			p.P(`i -= `, jvar)
			p.P(`copy(dAtA[i:], `, datavar, `[:`, jvar, `])`)
			p.callVarint(jvar)
			p.encodeKey(fieldNumber, wireType)
		} else if repeated {
			val := p.reverseListRange(`m.`, fieldname)
			p.P(`x`, numGen.Next(), ` := (uint32(`, val, `) << 1) ^ (uint32(`, val, `) >> 31)`)
			p.callVarint(`x`, numGen.Current())
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else if proto3 && !mustBeIncluded {
			p.P(`if m.`, fieldname, ` != 0 {`)
			p.In()
			p.callVarint(`(uint32(m.`, fieldname, `) << 1) ^ (uint32(m.`, fieldname, `) >> 31)`)
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else if !nullable {
			p.callVarint(`(uint32(m.`, fieldname, `) << 1) ^ (uint32(m.`, fieldname, `) >> 31)`)
			p.encodeKey(fieldNumber, wireType)
		} else {
			p.callVarint(`(uint32(*m.`, fieldname, `) << 1) ^ (uint32(*m.`, fieldname, `) >> 31)`)
			p.encodeKey(fieldNumber, wireType)
		}
	case descriptor.FieldDescriptorProto_TYPE_SINT64:
		if packed {
			jvar := "j" + numGen.Next()
			xvar := "x" + numGen.Next()
			datavar := "dAtA" + numGen.Next()
			p.P(`var `, jvar, ` int`)
			p.P(datavar, ` := make([]byte, len(m.`, fieldname, `)*10)`)
			p.P(`for _, num := range m.`, fieldname, ` {`)
			p.In()
			p.P(xvar, ` := (uint64(num) << 1) ^ (uint64(num) >> 63)`)
			p.P(`for `, xvar, ` >= 1<<7 {`)
			p.In()
			p.P(datavar, `[`, jvar, `] = uint8(uint64(`, xvar, `)|0x80)`)
			p.P(jvar, `++`)
			p.P(xvar, ` >>= 7`)
			p.Out()
			p.P(`}`)
			p.P(datavar, `[`, jvar, `] = uint8(`, xvar, `)`)
			p.P(jvar, `++`)
			p.Out()
			p.P(`}`)
			p.P(`i -= `, jvar)
			p.P(`copy(dAtA[i:], `, datavar, `[:`, jvar, `])`)
			p.callVarint(jvar)
			p.encodeKey(fieldNumber, wireType)
		} else if repeated {
			val := p.reverseListRange(`m.`, fieldname)
			p.P(`x`, numGen.Next(), ` := (uint64(`, val, `) << 1) ^ (uint64(`, val, `) >> 63)`)
			p.callVarint("x" + numGen.Current())
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else if proto3 && !mustBeIncluded {
			p.P(`if m.`, fieldname, ` != 0 {`)
			p.In()
			p.callVarint(`(uint64(m.`, fieldname, `) << 1) ^ (uint64(m.`, fieldname, `) >> 63)`)
			p.encodeKey(fieldNumber, wireType)
			p.Out()
			p.P(`}`)
		} else if !nullable {
			p.callVarint(`(uint64(m.`, fieldname, `) << 1) ^ (uint64(m.`, fieldname, `) >> 63)`)
			p.encodeKey(fieldNumber, wireType)
		} else {
			p.callVarint(`(uint64(*m.`, fieldname, `) << 1) ^ (uint64(*m.`, fieldname, `) >> 63)`)
			p.encodeKey(fieldNumber, wireType)
		}
	default:
		panic("not implemented")
	}
	if (required && nullable) || repeated || doNilCheck {
		p.Out()
		p.P(`}`)
	}
	if needsMapping {
		p.P(`if fieldEnd != i { m.FieldMap.Put(`, int(fieldNumber), `, i, fieldEnd, dAtA) }`)
	}
}

func (p *marshalto) generatePolymorphField(proto3, zeroableDefault bool, message *generator.Descriptor, field *descriptor.FieldDescriptorProto) {
	wireType := 0
	fieldType := descriptor.FieldDescriptorProto_TYPE_UINT64
	polymorphID := insproto.GetPolymorphID(message.DescriptorProto)
	polymorphArg := ""
	fieldName := "<polymorph>"
	hasIf := false

	if field == nil {
		polymorphArg = strconv.FormatUint(polymorphID, 10)
		if polymorphID == 0 {
			p.P(`if i < len(dAtA) {`)
			p.In()
			hasIf = true
		}
	} else {
		switch {
		case gogoproto.IsNullable(field):
			panic(throw.IllegalState())
		case field.IsRepeated():
			panic(throw.IllegalState())
		case field.IsPacked() || (proto3 && field.IsPacked3()):
			panic(throw.IllegalState())
		case field.GetDefaultValue() != "":
			panic(throw.IllegalState())
		}
		switch wireType = field.WireType(); wireType {
		case 0 /* varint */, 1 /* fixed64 */, 5 /* fixed32 */ :
		default:
			panic(throw.IllegalState())
		}

		zeroable := insproto.IsZeroable(field, zeroableDefault)
		fieldType = *field.Type
		fieldName = p.GetOneOfFieldName(message, field)
		if polymorphID != 0 {
			if zeroable {
				p.P(`if i < len(dAtA) {`)
			} else {
				p.P(`{`)
			}
			p.In()
			p.P(`id := uint64(m.`, fieldName, `)`)
			p.P(`if id == 0 { id = `, strconv.FormatUint(polymorphID, 10), ` }`)
			polymorphArg = `id`
			hasIf = true
		} else {
			if zeroable {
				p.P(`if i < len(dAtA) {`)
				p.In()
				hasIf = true
			}
			polymorphArg = `m.` + fieldName
		}
	}

	switch fieldType {
	case descriptor.FieldDescriptorProto_TYPE_INT64,
		descriptor.FieldDescriptorProto_TYPE_UINT64,
		descriptor.FieldDescriptorProto_TYPE_INT32,
		descriptor.FieldDescriptorProto_TYPE_UINT32,
		descriptor.FieldDescriptorProto_TYPE_ENUM:

		p.callVarint(polymorphArg)

	case descriptor.FieldDescriptorProto_TYPE_FIXED64,
		descriptor.FieldDescriptorProto_TYPE_SFIXED64:

		p.callFixed64(polymorphArg)

	case descriptor.FieldDescriptorProto_TYPE_FIXED32,
		descriptor.FieldDescriptorProto_TYPE_SFIXED32:

		p.callFixed32(polymorphArg)

	case descriptor.FieldDescriptorProto_TYPE_SINT32:
		p.callVarint(`(uint32(`, polymorphArg, `) << 1) ^ (uint32(`, polymorphArg, `) >> 31)`)

	case descriptor.FieldDescriptorProto_TYPE_SINT64:
		p.callVarint(`(uint64(`, polymorphArg, `) << 1) ^ (uint64(`, polymorphArg, `) >> 63)`)

	default:
		panic(fmt.Errorf("marshaler does not support type (%d) for %v", fieldType, fieldName))
	}

	p.encodeKey(16, wireType)
	if hasIf {
		p.Out()
		p.P(`}`)
	}
}

func (p *marshalto) GenerateImports(file *generator.FileDescriptor) {
	p.PluginImports.GenerateImports(file)
}

func (p *marshalto) Generate(file *generator.FileDescriptor) {
	p.PluginImports = generator.NewPluginImports(p.Generator)

	numGen := marshal.NewNumGen()

	p.atleastOne = false
	p.localName = generator.FileName(file)

	p.mathPkg = p.NewImport("math")
	p.sortKeysPkg = p.NewImport("github.com/gogo/protobuf/sortkeys")
	p.protoPkg = p.NewImport("github.com/gogo/protobuf/proto")
	if !gogoproto.ImportsGoGoProto(file.FileDescriptorProto) {
		p.protoPkg = p.NewImport("github.com/golang/protobuf/proto")
	}
	p.errorsPkg = p.NewImport("errors")
	p.binaryPkg = p.NewImport("encoding/binary")
	p.typesPkg = p.NewImport("github.com/gogo/protobuf/types")
	p.fieldMapPkg = p.NewImport(insproto.FieldMapPackage)

	p.contextGen.Init(p.Generator, p.PluginImports)
	p.polyGen.Init(p.Generator, p.PluginImports)

	for _, message := range file.Messages() {
		if message.DescriptorProto.GetOptions().GetMapEntry() {
			continue
		}
		ccTypeName := generator.CamelCaseSlice(message.TypeName())

		hasMarshaler := gogoproto.IsMarshaler(file.FileDescriptorProto, message.DescriptorProto) ||
			gogoproto.IsUnsafeMarshaler(file.FileDescriptorProto, message.DescriptorProto)

		notation := insproto.IsNotation(file.FileDescriptorProto, message.DescriptorProto)
		zeroableDefault := insproto.IsZeroableDefault(file.FileDescriptorProto, message.DescriptorProto)

		isProjection := extra.IsMessageProjection(file.FileDescriptorProto, message)

		if !isProjection {
			p.projections.Generate(file, message, ccTypeName)
		}
		p.contextGen.Generate(file, message, ccTypeName)
		p.polyGen.GenerateMsg(file, message, ccTypeName, isProjection)

		if !hasMarshaler {
			continue
		}

		p.atleastOne = true

		p.P(`func (m *`, ccTypeName, `) Marshal() (dAtA []byte, err error) {`)
		p.In()
		if gogoproto.IsProtoSizer(file.FileDescriptorProto, message.DescriptorProto) {
			p.P(`size := m.ProtoSize()`)
		} else {
			p.P(`size := m.Size()`)
		}
		p.P(`dAtA = make([]byte, size)`)
		p.P(`n, err := m.MarshalToSizedBuffer(dAtA[:size])`)
		p.P(`if err != nil {`)
		p.In()
		p.P(`return nil, err`)
		p.Out()
		p.P(`}`)
		p.P(`if n != size {`)
		p.In()
		p.P(`panic("illegal state")`)
		p.Out()
		p.P(`}`)
		p.P(`return dAtA[:n], nil`)
		p.Out()
		p.P(`}`)
		p.P(``)
		p.P(`func (m *`, ccTypeName, `) MarshalTo(dAtA []byte) (int, error) {`)
		p.In()
		if gogoproto.IsProtoSizer(file.FileDescriptorProto, message.DescriptorProto) {
			p.P(`size := m.ProtoSize()`)
		} else {
			p.P(`size := m.Size()`)
		}
		p.P(`return m.MarshalToSizedBuffer(dAtA[:size])`)
		p.Out()
		p.P(`}`)
		p.P(``)
		p.P(`func (m *`, ccTypeName, `) MarshalToSizedBuffer(dAtA []byte) (int, error) {`)
		p.In()
		p.P(`i := len(dAtA)`)
		p.P(`_ = i`)
		p.P(`var l, fieldEnd int`)
		p.P(`_, _ = l, fieldEnd`)
		if gogoproto.HasUnrecognized(file.FileDescriptorProto, message.DescriptorProto) {
			p.P(`if m.XXX_unrecognized != nil {`)
			p.In()
			p.P(`i -= len(m.XXX_unrecognized)`)
			p.P(`copy(dAtA[i:], m.XXX_unrecognized)`)
			p.Out()
			p.P(`}`)
		}
		if message.DescriptorProto.HasExtension() {
			if gogoproto.HasExtensionsMap(file.FileDescriptorProto, message.DescriptorProto) {
				p.P(`if n, err := `, p.protoPkg.Use(), `.EncodeInternalExtensionBackwards(m, dAtA[:i]); err != nil {`)
				p.In()
				p.P(`return 0, err`)
				p.Out()
				p.P(`} else {`)
				p.In()
				p.P(`i -= n`)
				p.Out()
				p.P(`}`)
			} else {
				p.P(`if m.XXX_extensions != nil {`)
				p.In()
				p.P(`i -= len(m.XXX_extensions)`)
				p.P(`copy(dAtA[i:], m.XXX_extensions)`)
				p.Out()
				p.P(`}`)
			}
		}

		fields := append([]*descriptor.FieldDescriptorProto(nil), message.GetField()...)

		fieldMapNo := int32(0)
		for i := len(fields) - 1; i >= 0; i-- {
			// FieldMap should be the last field
			field := fields[i]
			if field.GetName() == insproto.FieldMapFieldName && field.GetTypeName() == insproto.FieldMapFQN {
				fieldMapNo = field.GetNumber()
				p.P(`m.FieldMap.UnsetMap()`)
				break
			}
		}

		sort.Sort(extra.OrderedFields(fields))

		oneofs := make(map[string]struct{})
		protoSizer := gogoproto.IsProtoSizer(file.FileDescriptorProto, message.DescriptorProto)
		proto3 := gogoproto.IsProto3(file.FileDescriptorProto)

		mustBeIncluded := false
		addMissingPolymorph := false
		var polymorphField *descriptor.FieldDescriptorProto

		switch {
		case !notation:
			//
		case len(fields) == 0 || fields[0].GetNumber() > 19:
			addMissingPolymorph = true
		case fields[0].GetNumber() == 16:
			polymorphField = fields[0]
			fields = fields[1:]
		case insproto.HasPolymorphID(message.DescriptorProto):
			addMissingPolymorph = true
		default:
			mustBeIncluded = !isProjection
		}

		for i := len(fields) - 1; i >= 0; i-- {
			field := fields[i]

			fieldNum := field.GetNumber()
			if fieldNum == fieldMapNo {
				continue
			}

			fieldMustBeIncluded := mustBeIncluded && i == 0
			if fieldMustBeIncluded {
				if field.OneofIndex != nil || fieldNum < 16 || fieldNum > 19 {
					panic(fmt.Errorf("notation violation %v", ccTypeName))
				}
				mustBeIncluded = false
			}

			if field.OneofIndex == nil {
				p.generateField(proto3, notation, zeroableDefault, protoSizer, fieldMapNo > 0, fieldMustBeIncluded,
					numGen, file, message, field)
				continue
			}

			fieldname := p.GetFieldName(message, field)
			if _, ok := oneofs[fieldname]; ok {
				continue
			}
			oneofs[fieldname] = struct{}{}
			p.P(`if m.`, fieldname, ` != nil {`)
			p.In()
			p.forward(`m.`+fieldname, false, protoSizer)
			p.Out()
			p.P(`}`)
		}

		if addMissingPolymorph || polymorphField != nil {
			p.generatePolymorphField(proto3, zeroableDefault, message, polymorphField)
		}

		if fieldMapNo > 0 {
			p.P(`m.FieldMap.PutMessage(i, len(dAtA), dAtA)`)
		}

		p.P(`return len(dAtA) - i, nil`)
		p.Out()
		p.P(`}`)
		p.P()

		if fieldMapNo > 0 {
			p.P(`func (m *`, ccTypeName, `) InitFieldMap(reset bool) *`, p.fieldMapPkg.Use(), `.FieldMap {`)
			p.In()
			p.P(`if reset || m.FieldMap == nil {`)
			p.In()
			p.P(`m.FieldMap = &`, p.fieldMapPkg.Use(), `.FieldMap{}`)
			p.Out()
			p.P(`}`)
			p.P(`return m.FieldMap`)
			p.Out()
			p.P(`}`)
			p.P()
		}

		if len(oneofs) > 0 {
			// Generate MarshalTo methods for oneof fields
			m := proto.Clone(message.DescriptorProto).(*descriptor.DescriptorProto)
			for _, field := range m.Field {
				if field.OneofIndex == nil {
					continue
				}
				ccTypeName := p.OneOfTypeName(message, field)
				p.P(`func (m *`, ccTypeName, `) MarshalTo(dAtA []byte) (int, error) {`)
				p.In()
				if gogoproto.IsProtoSizer(file.FileDescriptorProto, message.DescriptorProto) {
					p.P(`size := m.ProtoSize()`)
				} else {
					p.P(`size := m.Size()`)
				}
				p.P(`return m.MarshalToSizedBuffer(dAtA[:size])`)
				p.Out()
				p.P(`}`)
				p.P(``)
				p.P(`func (m *`, ccTypeName, `) MarshalToSizedBuffer(dAtA []byte) (int, error) {`)
				p.In()
				p.P(`i := len(dAtA)`)
				vanity.TurnOffNullableForNativeTypes(field)
				p.generateField(false, notation, zeroableDefault, protoSizer, false, false, numGen, file, message, field)
				p.P(`return len(dAtA) - i, nil`)
				p.Out()
				p.P(`}`)
				p.P()
			}
		}
	}

	if p.atleastOne {
		p.P(`func encodeVarint`, p.localName, `(dAtA []byte, offset int, v uint64) int {`)
		p.In()
		p.P(`offset -= sov`, p.localName, `(v)`)
		p.P(`base := offset`)
		p.P(`for v >= 1<<7 {`)
		p.In()
		p.P(`dAtA[offset] = uint8(v&0x7f|0x80)`)
		p.P(`v >>= 7`)
		p.P(`offset++`)
		p.Out()
		p.P(`}`)
		p.P(`dAtA[offset] = uint8(v)`)
		p.P(`return base`)
		p.Out()
		p.P(`}`)
		p.P()
	}

	p.polyGen.GenerateFile()
}

func (p *marshalto) reverseListRange(expression ...string) string {
	exp := strings.Join(expression, "")
	p.P(`for iNdEx := len(`, exp, `) - 1; iNdEx >= 0; iNdEx-- {`)
	p.In()
	return exp + `[iNdEx]`
}

func (p *marshalto) marshalAllSizeOf(field *descriptor.FieldDescriptorProto, varName, num string) bool {
	if gogoproto.IsStdTime(field) {
		p.marshalSizeOf(`StdTimeMarshalTo`, `SizeOfStdTime`, varName, num)
	} else if gogoproto.IsStdDuration(field) {
		p.marshalSizeOf(`StdDurationMarshalTo`, `SizeOfStdDuration`, varName, num)
	} else if gogoproto.IsStdDouble(field) {
		p.marshalSizeOf(`StdDoubleMarshalTo`, `SizeOfStdDouble`, varName, num)
	} else if gogoproto.IsStdFloat(field) {
		p.marshalSizeOf(`StdFloatMarshalTo`, `SizeOfStdFloat`, varName, num)
	} else if gogoproto.IsStdInt64(field) {
		p.marshalSizeOf(`StdInt64MarshalTo`, `SizeOfStdInt64`, varName, num)
	} else if gogoproto.IsStdUInt64(field) {
		p.marshalSizeOf(`StdUInt64MarshalTo`, `SizeOfStdUInt64`, varName, num)
	} else if gogoproto.IsStdInt32(field) {
		p.marshalSizeOf(`StdInt32MarshalTo`, `SizeOfStdInt32`, varName, num)
	} else if gogoproto.IsStdUInt32(field) {
		p.marshalSizeOf(`StdUInt32MarshalTo`, `SizeOfStdUInt32`, varName, num)
	} else if gogoproto.IsStdBool(field) {
		p.marshalSizeOf(`StdBoolMarshalTo`, `SizeOfStdBool`, varName, num)
	} else if gogoproto.IsStdString(field) {
		p.marshalSizeOf(`StdStringMarshalTo`, `SizeOfStdString`, varName, num)
	} else if gogoproto.IsStdBytes(field) {
		p.marshalSizeOf(`StdBytesMarshalTo`, `SizeOfStdBytes`, varName, num)
	} else {
		return false
	}
	return true
}

func (p *marshalto) marshalSizeOf(marshal, size, varName, num string) {
	p.P(`n`, num, `, err`, num, ` := `, p.typesPkg.Use(), `.`, marshal, `(`, varName, `, dAtA[i-`, p.typesPkg.Use(), `.`, size, `(`, varName, `):])`)
	p.P(`if err`, num, ` != nil {`)
	p.In()
	p.P(`return 0, err`, num)
	p.Out()
	p.P(`}`)
	p.P(`i -= n`, num)
	p.callVarint(`n`, num)
}

func (p *marshalto) backward(varName string, varInt bool) {
	p.P(`{`)
	p.In()
	p.P(`size, err := `, varName, `.MarshalToSizedBuffer(dAtA[:i])`)
	p.P(`if err != nil {`)
	p.In()
	p.P(`return 0, err`)
	p.Out()
	p.P(`}`)
	p.P(`i -= size`)
	if varInt {
		p.callVarint(`size`)
	}
	p.Out()
	p.P(`}`)
}

func (p *marshalto) backwardZeroable(varName string, varInt bool, encodeKeyFn func()) {
	p.P(`{`)
	p.In()
	p.P(`size, err := `, varName, `.MarshalToSizedBuffer(dAtA[:i])`)
	p.P(`if err != nil {`)
	p.In()
	p.P(`return 0, err`)
	p.Out()
	p.P(`}`)

	if encodeKeyFn != nil {
		p.P(`if size > 0 {`)
		p.In()
	}

	p.P(`i -= size`)
	if varInt {
		p.callVarint(`size`)
	}

	if encodeKeyFn != nil {
		encodeKeyFn()
		p.Out()
		p.P(`}`)
	}

	p.Out()
	p.P(`}`)
}

func (p *marshalto) forward(varName string, varInt, protoSizer bool) {
	p.forwardZeroable(varName, varInt, protoSizer, true, nil)
}

func (p *marshalto) forwardZeroable(varName string, varInt, protoSizer, isRaw bool, encodeKeyFn func()) {
	p.P(`{`)
	p.In()
	if protoSizer {
		p.P(`size := `, varName, `.ProtoSize()`)
	} else {
		p.P(`size := `, varName, `.Size()`)
	}

	if encodeKeyFn != nil {
		p.P(`if size > 0 {`)
		p.In()
	}

	p.P(`i -= size`)
	p.P(`if _, err := `, varName, `.MarshalTo(dAtA[i:]); err != nil {`)
	p.In()
	p.P(`return 0, err`)
	p.Out()
	p.P(`}`)
	p.Out()
	if varInt {
		if isRaw {
			p.callVarint(`size`)
		} else {
			p.P(`i--`)
			p.P(`dAtA[i]=`, int(protokit.BinaryMarker))
			p.callVarint(`size+1`)
		}
	}

	if encodeKeyFn != nil {
		encodeKeyFn()
		p.Out()
		p.P(`}`)
	}

	p.P(`}`)
}
