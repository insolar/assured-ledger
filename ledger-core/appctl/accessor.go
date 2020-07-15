// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package appctl

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl.Accessor -o ./ -s _mock.go -g

// Accessor provides methods for accessing pulses.
type Accessor interface {
	ForPulseNumber(context.Context, pulse.Number) (PulseChange, error)
	Latest(ctx context.Context) (PulseChange, error)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl.Appender -o ./ -s _mock.go -g

// Appender provides method for appending pulses to storage.
type Appender interface {
	Append(ctx context.Context, pulse PulseChange) error
}

