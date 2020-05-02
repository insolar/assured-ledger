// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

type ByteString = longbits.ByteString
type RecordBody = BlobBody
type RecordExtension = BlobBody
type PulseNumber = pulse.Number

type RecordContext interface {
	Record(BasicRecord) error
}

type BasicRecord interface {
	GetDefaultPolymorphID() uint64
	SetupContext(RecordContext) error
	GetFieldMap() insproto.FieldMap
	GetBodyHash() RecordBody
	GetExtensionHash() []RecordExtension
}

type MessageContext interface {
	Message(BasicMessage) error
	MsgRecord(BasicMessage, int, BasicRecord) error
}

type BasicMessage interface {
	GetDefaultPolymorphID() uint64
	SetupContext(MessageContext) error
}

func RegisterRecordType(t BasicRecord) {
	t.GetDefaultPolymorphID()
}

func RegisterMessageType(t BasicMessage) {
	t.GetDefaultPolymorphID()
}
