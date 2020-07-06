// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"reflect"

	"github.com/gogo/protobuf/proto"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type PulseNumber = pulse.Number

type RecordVisitor interface {
	Record(BasicRecord, uint64) error
	RecReference(BasicRecord, uint64, *Reference) error
}

type BasicRecord interface {
	Visit(RecordVisitor) error
	GetRecordPayloads() RecordPayloads
	SetRecordPayloads(RecordPayloads, cryptkit.DataDigester) error
}

type MessageVisitor interface {
	Message(BasicMessage, uint64) error
	MsgRecord(BasicMessage, int, BasicRecord) error
}

type BasicMessage interface {
	Visit(MessageVisitor) error
}

func RegisterRecordType(id uint64, special string, t BasicRecord) {
	GetRegistry().PutSpecial(id, special, reflect.TypeOf(t))
}

func RegisterMessageType(id uint64, special string, t proto.Message) {
	GetRegistry().PutSpecial(id, special, reflect.TypeOf(t))
}
