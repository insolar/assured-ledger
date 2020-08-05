// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package beat

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/beat.Accessor -o ./ -s _mock.go -g

// Accessor provides methods for accessing pulses.
type Accessor interface {
	TimeBeat(pulse.Number) (Beat, error)
	LatestTimeBeat() (Beat, error)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/beat.Appender -o ./ -s _mock.go -g

// Appender provides method for appending pulses to storage.
type Appender interface {
	Accessor
	AddCommittedBeat(Beat) error
	EnsureLatest(Beat) error
}

