// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datareader

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type ExtractedRecord = rms.LReadResponse_Entry
type ExtractedTail struct {
	NextRecordSize, NextRecordPayloadsSize int
}

type SequenceExtractor interface {
	AddRecord(lineage.ReadRecord) bool
	NeedsDirtyReader() bool

	ExtractAllRecordsWithReader(bundle.DirtyReader)
	ExtractMoreRecords(batchCount int) bool

	GetExtractRecords() []ExtractedRecord
	GetExtractedTail() ExtractedTail
}

type SequenceLimiter interface {

}

type ExtractionType uint8

const (
	ExtractNone ExtractionType = iota
	ExtractProof
	ExtractExcerpt

	// Redirection support?

	ExtractRecord
	ExtractFull
)

type ExtractorStrategy interface {

}
