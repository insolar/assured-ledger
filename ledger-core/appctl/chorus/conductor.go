package chorus

import (
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/appctl/chorus.Conductor -s _mock.go -g
// Conductor provides methods to orchestrate pulses through application's components.
type Conductor interface {
	NodeStater
	
	// Set set's new pulse and closes current jet drop.
	CommitPulseChange(beat.Beat) error
	CommitFirstPulseChange(beat.Beat) error
}

type NodeStateFunc = func(api.UpstreamState)

type NodeStater interface {
	RequestNodeState(NodeStateFunc)
	CancelNodeState()
}
