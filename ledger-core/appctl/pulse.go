// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package appctl

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl.PulseAccessor -s _mock.go -g

// PulseAccessor provides methods for accessing pulses.
type PulseAccessor interface {
	GetPulse(context.Context, pulse.Number) (PulseChange, error)
	GetLatestPulse(context.Context) (PulseChange, error)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl.PulseAppender -s _mock.go -g

// PulseAppender provides method for appending pulses to storage.
type PulseAppender interface {
	AppendPulse(context.Context, PulseChange) error
}
