package pulsar

import (
	"context"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/pulsar.PulseDistributor -s _mock.go -g

// PulseDistributor is interface for pulse distribution.
type PulseDistributor interface {
	// Distribute distributes a pulse across the network.
	Distribute(context.Context, PulsePacket)
}
