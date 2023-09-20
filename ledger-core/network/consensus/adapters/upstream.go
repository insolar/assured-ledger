package adapters

import (
	"context"
	"crypto/rand"
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/appctl/chorus"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/censusimpl"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type NodeStater interface {
	RequestNodeState(chorus.NodeStateFunc)
	CancelNodeState()
}

type BeatChanger interface {
	ChangeBeat(context.Context, api.UpstreamReport, beat.Beat)
}

type StateUpdater interface {
	UpdateState(context.Context, beat.Beat)
}

type UpstreamController struct {
	stateGetter  NodeStater
	beatChanger  BeatChanger
	stateUpdater StateUpdater

	mu         *sync.RWMutex
	onFinished network.OnConsensusFinished
}

func NewUpstreamPulseController(stateGetter NodeStater, pulseChanger BeatChanger, stateUpdater StateUpdater) *UpstreamController {
	return &UpstreamController{
		stateGetter:  stateGetter,
		beatChanger:  pulseChanger,
		stateUpdater: stateUpdater,

		mu:         &sync.RWMutex{},
		onFinished: func(ctx context.Context, report network.Report) {},
	}
}

func (u *UpstreamController) ConsensusFinished(report api.UpstreamReport, expectedCensus census.Operational) {
	ctx := ReportContext(report)
	logger := inslogger.FromContext(ctx)
	population := expectedCensus.GetOnlinePopulation()

	// TODO this mimics legacy code. Must be changed
	switch {
	case report.MemberMode.IsEvicted() || !population.IsValid():
		pop := censusimpl.CopySelfNodePopulation(population)
		population = &pop
		fallthrough
	case report.MemberMode.IsSuspended():
		logger.Warnf("Consensus finished unexpectedly mode: %s, population: %v", report.MemberMode, expectedCensus)
	}

	_, pd := expectedCensus.GetNearestPulseData()
	u.stateUpdater.UpdateState(ctx, beat.Beat{
		BeatSeq:     0,
		Data:        pd,
		Online:      population,
	})

	u.mu.RLock()
	defer u.mu.RUnlock()

	u.onFinished(ctx, network.Report{
		PulseData:       pd,
		PulseNumber:     report.PulseNumber,
		MemberPower:     report.MemberPower,
		MemberMode:      report.MemberMode,
		IsJoiner:        report.IsJoiner,
		PopulationValid: population.IsValid(),
	})
}

func (u *UpstreamController) ConsensusAborted() {
	// TODO implement
}

func (u *UpstreamController) PreparePulseChange(_ api.UpstreamReport, ch chan<- api.UpstreamState) {
	// is only called on non-ephemeral pulse
	u.stateGetter.RequestNodeState(func(state api.UpstreamState) {
		switch {
		case state.NodeState == nil:
			nshBytes := longbits.Bits512{}
			_, _ = rand.Read(nshBytes[:])
			ch <- api.UpstreamState{NodeState: cryptkit.NewDigest(nshBytes, "random")}

		case state.NodeState.FixedByteSize() != 64:
			panic(throw.IllegalState())
		default:
			ch <- state
		}
	})
}

func (u *UpstreamController) CancelPulseChange() {
	// is only called on non-ephemeral pulse
	u.stateGetter.CancelNodeState()
}

func (u *UpstreamController) CommitPulseChange(report api.UpstreamReport, pulseData pulse.Data, activeCensus census.Operational) {
	ctx := ReportContext(report)
	online := activeCensus.GetOnlinePopulation()

	u.beatChanger.ChangeBeat(ctx, report, beat.Beat{
		BeatSeq:   0, // TODO move into report?
		Data:      pulseData,
		StartedAt: time.Now(), // TODO get pulse start
		Online:    online,
	})
}

func (u *UpstreamController) SetOnFinished(f network.OnConsensusFinished) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.onFinished = f
}
