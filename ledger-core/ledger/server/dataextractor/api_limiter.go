package dataextractor

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type Limiter interface {
	CanRead() bool
	IsSkipped(recType uint32) bool
	CanReadExcerpt() bool
	CanReadBody() bool
	CanReadPayload() bool
	CanReadExtensions() bool
	CanReadExtension(ledger.ExtensionID) bool

	Next(consumedSize int, nextRecRef reference.Holder)

	Clone() Limiter
}

type Limits struct {
	StopRef   reference.Global
	TotalSize uint64

	TotalCount uint32
	Excerpts   uint32
	Bodies     uint32
	Payloads   uint32
	Extensions uint32

	RecTypeFilter func(uint32) bool
	ExtTypeFilter func(ledger.ExtensionID) bool

	ExcludeStart bool
	ExcludeStop  bool
}
