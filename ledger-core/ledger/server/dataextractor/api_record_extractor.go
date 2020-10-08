// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dataextractor

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc/readbundle"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type ExtractedRecord = rms.LReadResponse_Entry

type ExtractedTail struct {
	NextRecordSize, NextRecordPayloadsSize int
}

type SelectedRecord struct {
	Index     ledger.DirectoryIndex
	RecordRef reference.Holder
	MinSize   int
}

type SequenceExtractor interface {
	AddExpectedRecord(SelectedRecord) bool
	NeedsReader() bool

	ExtractRecordsWithReader(readbundle.BasicReader) error

	GetExtractedRecords() []ExtractedRecord
	GetExtractedTail() ExtractedTail
}
