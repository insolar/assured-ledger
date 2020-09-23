// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsar

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/reference"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/pulsar.PulseDistributor -s _mock.go -g

// PulseDistributor is interface for pulse distribution.
type PulseDistributor interface {
	// Distribute distributes a pulse across the network.
	Distribute(context.Context, PulsePacket)
}

// SelectivePulseDistributor is interface for selective pulse distribution.
type SelectivePulseDistributor interface {
	// PartialDistribute distributes a pulse basing on white list.
	PartialDistribute(context.Context, PulsePacket, map[reference.Global]struct{})
}
