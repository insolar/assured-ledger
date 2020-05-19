// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulse

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse.Accessor -o ./ -s _mock.go -g

// Accessor provides methods for accessing pulses.
type Accessor interface {
	ForPulseNumber(context.Context, pulse.Number) (Pulse, error)
	Latest(ctx context.Context) (Pulse, error)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse.Shifter -o ./ -s _mock.go -g

// Shifter provides method for removing pulses from storage.
type Shifter interface {
	Shift(ctx context.Context, pn pulse.Number) (err error)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse.Appender -o ./ -s _mock.go -g

// Appender provides method for appending pulses to storage.
type Appender interface {
	Append(ctx context.Context, pulse Pulse) error
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse.Calculator -o ./ -s _mock.go -g

// Calculator performs calculations for pulses.
type Calculator interface {
	Forwards(ctx context.Context, pn pulse.Number, steps int) (Pulse, error)
	Backwards(ctx context.Context, pn pulse.Number, steps int) (Pulse, error)
}
