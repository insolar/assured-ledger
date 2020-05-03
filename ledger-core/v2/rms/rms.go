// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/v2/insproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

type RecordExtension = BodyWithDigest
type RecordBody = BodyWithDigest

type ByteString = longbits.ByteString
type PulseNumber = pulse.Number

type RecordContext interface {
	Record(BasicRecord, uint64) error
}

type BasicRecord interface {
	SetupContext(RecordContext) error
	GetFieldMap() *insproto.FieldMap
	GetBody() *RecordBody
	GetExtension() []RecordExtension
}

type MessageContext interface {
	Message(BasicMessage, uint64) error
	MsgRecord(BasicMessage, int, BasicRecord) error
}

type BasicMessage interface {
	SetupContext(MessageContext) error
}

func RegisterRecordType(id uint64, t BasicRecord) {
	GetRegistry().Put(id, reflect.TypeOf(t))
}

func RegisterMessageType(id uint64, t BasicMessage) {
	GetRegistry().Put(id, reflect.TypeOf(t))
}
