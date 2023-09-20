package termination

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network/termination.Leaver -s _mock.go -g

type Leaver interface {
	// Leave notify other nodes that this node want to leave and doesn't want to receive new tasks
	Leave(ctx context.Context, ETA pulse.Number)
}
