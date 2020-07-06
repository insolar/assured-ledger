// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cabinet

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type BatchID uint32
type BatchIDFeed <-chan BatchID

type Writer interface {
	// known SectionID
	CommitConfirmations() BatchIDFeed
	StartBatch(jet.DropID) EntryBatchWriter
	TruncateWrite() BatchID
}

type EntryBatchWriter interface {
	BatchID() BatchID
	CommitBatch()
//	RollbackBatch()
	WriteEntry(SectionID, reference.Holder, func(EntryWriter)) bool
}

type MarshalerTo interface {
	ProtoSize() int
	MarshalTo([]byte) (int, error)
}

type EntryWriter interface {
	DirectoryIndex() DirectoryIndex
	WritePayload(SectionID, MarshalerTo) StorageLocator
	WriteEntry(MarshalerTo) StorageLocator
}

