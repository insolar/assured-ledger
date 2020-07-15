// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsestor

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/appctl"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor.Calculator -o ./ -s _mock.go -g

// Calculator performs calculations for pulses.
type Calculator interface {
	Backwards(ctx context.Context, pn pulse.Number, steps int) (appctl.PulseChange, error)
}
