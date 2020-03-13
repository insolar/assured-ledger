/*
 * Copyright 2020 Insolar Network Ltd.
 * All rights reserved.
 * This material is licensed under the Insolar License version 1.0,
 * available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package rms

//go:generate protoc -I=. -I=$GOPATH/src --gogofaster_out=./ example.proto

type MsgHeadExample struct {
	MsgExample
}

var _ RecordHolder = &RecExample{}

func (m *RecExample) Head() GoGoMessage {
	return m
}

func (m *RecExample) Body() *RecordBodyHolder {
	return &m.BodyHash
}

func (m *RecExample) EntryExtractor() RecordEntryExtractorFunc {
	panic("implement me")
}

var _ MessageHolder = &MsgExample{}

func (m *MsgExample) MsgBody() GoGoMessage {
	return m
}

func (m *MsgExample) PrimaryRecord() RecordHolder {
	return &m.RecExample
}
