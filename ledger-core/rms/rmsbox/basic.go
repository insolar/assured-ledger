package rmsbox

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type RecordVisitor interface {
	Record(BasicRecord, uint64) error
	RecReference(BasicRecord, uint64, *Reference) error
}

type PayloadHolder interface {
	GetRecordPayloads() RecordPayloads

	// SetRecordPayloads is called after unmarshalling of the record to set content of record's payloads
	SetRecordPayloads(RecordPayloads, cryptkit.DataDigester) error
}

type BasicRecord interface {
	PayloadHolder
	Visit(RecordVisitor) error
}

type MessageVisitor interface {
	Message(BasicMessage, uint64) error
	MsgRecord(BasicMessage, int, BasicRecord) error
}

type BasicMessage interface {
	// PayloadHolder
	Visit(MessageVisitor) error
}
