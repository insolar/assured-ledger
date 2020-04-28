// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

//go:generate protoc -I=. -I=$GOPATH/src --gogofaster_out=./ example.proto

type MsgHeadExample struct {
	MsgExample
}

/*********************************************************/
var _ RecordHolder = &RecExample{}

func (m *RecExample) Head() GoGoMarshaller {
	return m
}

func (m *RecExample) Body() *RecordBodyHolder {
	return &m.BodyHash
}

func (m *RecExample) EntryExtractor() RecordEntryExtractorFunc {
	panic("implement me")
}

func (m *RecExample) InitPolymorphField(setup bool) bool {
	const polymorphId = 100
	if setup {
		m.Polymorph = polymorphId
		return true
	}
	return m.Polymorph == polymorphId
}

/*********************************************************/
var _ RecordHolder = &RecExample2{}

func (m *RecExample2) Head() GoGoMarshaller {
	return m
}

func (m *RecExample2) Body() *RecordBodyHolder {
	return &m.BodyHash
}

func (m *RecExample2) EntryExtractor() RecordEntryExtractorFunc {
	panic("implement me")
}

func (m *RecExample2) InitPolymorphField(setup bool) bool {
	const polymorphId = 102
	if setup {
		m.Polymorph = polymorphId
		return true
	}
	return m.Polymorph == polymorphId
}

/*********************************************************/
var _ MessageHolder = &MsgExample{}

func (m *MsgExample) MsgBody() GoGoMessage {
	return m
}

func (m *MsgExample) PrimaryRecord() RecordHolder {
	return &m.RecExample
}

func (m *MsgExample) InitPolymorphField(setup bool) bool {
	const polymorphId = 101
	if setup {
		m.MsgPolymorph = polymorphId
		return true
	}
	return m.MsgPolymorph == polymorphId
}

func init() {
	RegisterPolymorph(100, func() ProtoMessage { return &RecExample{} })
	RegisterPolymorph(101, func() ProtoMessage { return &MsgExample{} })
	RegisterPolymorph(102, func() ProtoMessage { return &RecExample2{} })
}
