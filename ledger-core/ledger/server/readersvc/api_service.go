// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package readersvc

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/readersvc/readbundle"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type Service interface {
	FindCabinet(pulse.Number) (Cabinet, error)
	ReadFromCabinet(Cabinet, jet.DropID, func(readbundle.Reader) error) error

	// NeedsBatching is (true) when readers must be organized in batches, also FindCabinet will return error while the cabinet is opened.
	// When NeedsBatching is (false) then every reader can open cabinets independently.
	NeedsBatching() bool
}

type Cabinet interface {
	PulseNumber() pulse.Number

	Open() error
	io.Closer
}
