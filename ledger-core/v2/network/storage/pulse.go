// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package storage

import (
	"context"

	pulse2 "github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/storage.PulseAccessor -o ../../testutils/network -s _mock.go -g

// PulseAccessor provides methods for accessing pulses.
type PulseAccessor interface {
	GetPulse(context.Context, pulse.Number) (pulse2.Pulse, error)
	GetLatestPulse(ctx context.Context) (pulse2.Pulse, error)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/storage.PulseAppender -o ../../testutils/network -s _mock.go -g

// PulseAppender provides method for appending pulses to storage.
type PulseAppender interface {
	AppendPulse(ctx context.Context, pulse pulse2.Pulse) error
}
