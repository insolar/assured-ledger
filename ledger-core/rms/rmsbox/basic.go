// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rmsbox

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type RecordVisitor interface {
	Record(BasicRecord, uint64) error
	RecReference(BasicRecord, uint64, *Reference) error
}

type BasicRecord interface {
	Visit(RecordVisitor) error
	GetRecordPayloads() RecordPayloads

	// SetRecordPayloads is called after unmarshalling of the record to set content of record's payloads
	SetRecordPayloads(RecordPayloads, cryptkit.DataDigester) error
}

type MessageVisitor interface {
	Message(BasicMessage, uint64) error
	MsgRecord(BasicMessage, int, BasicRecord) error
}

type BasicMessage interface {
	Visit(MessageVisitor) error
}
