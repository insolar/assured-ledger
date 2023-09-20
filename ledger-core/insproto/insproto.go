package insproto

import (
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const FieldMapFQN = `.insproto.FieldMap`
const FieldMapFieldName = `FieldMap`
const FieldMapPackage = `github.com/insolar/assured-ledger/ledger-core/insproto`

type FieldMapCallback interface {
	OnMessage(*FieldMap)
}

type FieldMap struct {
	Message  []byte
	Fields   map[int32][]byte
	Callback FieldMapCallback
}

func (p *FieldMap) PutMessage(msgStart, msgEnd int, b []byte) {
	if p == nil {
		return
	}
	p.Message = b[msgStart:msgEnd]
	if p.Callback != nil {
		p.Callback.OnMessage(p)
	}
}

func (p *FieldMap) Put(fieldNum int32, fieldStart, fieldEnd int, data []byte) {
	if p == nil {
		return
	}
	if p.Fields == nil {
		p.Fields = map[int32][]byte{}
	}
	p.Fields[fieldNum] = data[fieldStart:fieldEnd]
}

func (p *FieldMap) Get(fieldNum int32) []byte {
	if p == nil {
		return nil
	}
	return p.Fields[fieldNum]
}

func (p *FieldMap) GetMessage() []byte {
	if p == nil {
		return nil
	}
	return p.Message
}

func (*FieldMap) MarshalTo([]byte) (int, error) {
	return 0, nil
}

func (*FieldMap) MarshalToSizedBuffer([]byte) (int, error) {
	return 0, nil
}

func (*FieldMap) Unmarshal([]byte) error {
	return throw.Impossible()
}

func (*FieldMap) Equal(*FieldMap) bool {
	return true
}

func (p *FieldMap) UnsetMap() {
	if p == nil {
		return
	}
	p.Message = nil
	p.Fields = nil
}

func NewFieldMapDescriptorProto(number int32) *descriptor.FieldDescriptorProto {
	name := FieldMapFieldName
	typeName := FieldMapFQN
	label := descriptor.FieldDescriptorProto_LABEL_OPTIONAL
	fieldType := descriptor.FieldDescriptorProto_TYPE_MESSAGE
	if number == 0 {
		number = 19999 // reserved by proto
	}
	return &descriptor.FieldDescriptorProto{
		Name:     &name,
		TypeName: &typeName,
		Type:     &fieldType,
		Label:    &label,
		Number:   &number,
	}
}
