// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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

