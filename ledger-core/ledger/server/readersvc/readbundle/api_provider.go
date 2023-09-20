package readbundle

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type Provider interface {
	FindCabinet(pulse.Number) (ReadCabinet, error)
	// LastStorage, FirstStorage
}

type ReadCabinet interface {
	PulseNumber() pulse.Number

	Open() error
	Reader() Reader
	io.Closer
}

