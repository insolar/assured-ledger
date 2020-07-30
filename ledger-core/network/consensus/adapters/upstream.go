// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
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
	UpdateState(ctx context.Context, pulseNumber pulse.Number, nodes []nodeinfo.NetworkNode, cloudStateHash []byte)
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

	var networkNodes []nodeinfo.NetworkNode
	if report.MemberMode.IsEvicted() || report.MemberMode.IsSuspended() || !population.IsValid() {
		logger.Warnf("Consensus finished unexpectedly mode: %s, population: %v", report.MemberMode, expectedCensus)

		networkNodes = []nodeinfo.NetworkNode{
			nodeinfo.NewNetworkNode(expectedCensus.GetOnlinePopulation().GetLocalProfile()),
		}
	} else {
		networkNodes = nodeinfo.NewNetworkNodeList(population.GetProfiles())
	}

	u.stateUpdater.UpdateState(
		ctx,
		report.PulseNumber, // not used
		networkNodes,
		longbits.AsBytes(expectedCensus.GetCloudStateHash()), // not used
	)

	// todo: ??
	_, pd := expectedCensus.GetNearestPulseData()
	if pd.IsFromEphemeral() {
		// Fix bootstrap. Commit active list right after consensus finished
		// for NodeKeeper active list move sync to active
		u.CommitPulseChange(report, pd, expectedCensus)
	}

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
