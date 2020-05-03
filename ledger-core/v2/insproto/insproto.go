// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insproto

import (
	"github.com/gogo/protobuf/protoc-gen-gogo/descriptor"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

const FieldMapFQN = `.insproto.FieldMap`
const FieldMapFieldName = `FieldMap`
const FieldMapPackage = `github.com/insolar/assured-ledger/ledger-core/v2/insproto`

type FieldMapCallback interface {
	OnMessage(*FieldMap)
}

type FieldMap struct {
	Message  []byte
	Fields   map[int32][]byte
	Callback FieldMapCallback
}

func (p *FieldMap) PutMessage(b []byte) {
	if p == nil {
		return
	}
	p.Message = b
	if p.Callback != nil {
		p.Callback.OnMessage(p)
	}
}

func (p *FieldMap) Put(fieldNum int32, fieldSlice []byte) {
	if p == nil {
		return
	}
	if p.Fields == nil {
		p.Fields = map[int32][]byte{}
	}
	p.Fields[fieldNum] = fieldSlice
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

func (FieldMap) MarshalTo([]byte) (int, error) {
	return 0, nil
}

func (FieldMap) MarshalToSizedBuffer([]byte) (int, error) {
	return 0, nil
}

func (FieldMap) Unmarshal([]byte) error {
	return throw.Impossible()
}

func (FieldMap) Equal(*FieldMap) bool {
	return true
}

func NewFieldMapDescriptorProto(number int32) *descriptor.FieldDescriptorProto {
	name := FieldMapFieldName
	typeName := FieldMapFQN
	label := descriptor.FieldDescriptorProto_LABEL_OPTIONAL
	fieldType := descriptor.FieldDescriptorProto_TYPE_MESSAGE
	if number == 0 {
		number = 999999
	}
	return &descriptor.FieldDescriptorProto{
		Name:     &name,
		TypeName: &typeName,
		Type:     &fieldType,
		Label:    &label,
		Number:   &number,
	}
}
