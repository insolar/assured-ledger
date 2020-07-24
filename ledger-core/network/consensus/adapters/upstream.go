// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapters

import (
	"context"
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

type StateGetter interface {
	State() []byte
}

type BeatChanger interface {
	ChangeBeat(context.Context, beat.Beat)
}

type StateUpdater interface {
	UpdateState(ctx context.Context, pulseNumber pulse.Number, nodes []nodeinfo.NetworkNode, cloudStateHash []byte)
}

type UpstreamController struct {
	stateGetter  StateGetter
	beatChanger  BeatChanger
	stateUpdater StateUpdater

	mu         *sync.RWMutex
	onFinished network.OnConsensusFinished
}

func NewUpstreamPulseController(stateGetter StateGetter, pulseChanger BeatChanger, stateUpdater StateUpdater) *UpstreamController {
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
	// TODO::
	// expectedCensus.

	var networkNodes []nodeinfo.NetworkNode
	if report.MemberMode.IsEvicted() || report.MemberMode.IsSuspended() || !population.IsValid() {
		logger.Warnf("Consensus finished unexpectedly mode: %s, population: %v", report.MemberMode, expectedCensus)

		networkNodes = []nodeinfo.NetworkNode{
			NewNetworkNode(expectedCensus.GetOnlinePopulation().GetLocalProfile()),
		}
	} else {
		networkNodes = NewNetworkNodeList(population.GetProfiles())
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

func (u *UpstreamController) PreparePulseChange(report api.UpstreamReport, ch chan<- api.UpstreamState) {
	go awaitState(ch, u.stateGetter)
}

func (u *UpstreamController) CommitPulseChange(report api.UpstreamReport, pulseData pulse.Data, activeCensus census.Operational) {
	ctx := ReportContext(report)
	online := activeCensus.GetOnlinePopulation()

	// todo
	u.beatChanger.ChangeBeat(ctx, beat.Beat{
		BeatSeq:   0,
		Data:      pulseData,
		StartedAt: time.Now(), // TODO get pulse start
		Online:    online,
	})
}

func (u *UpstreamController) CancelPulseChange() {
	// TODO implement
}

func (u *UpstreamController) SetOnFinished(f network.OnConsensusFinished) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.onFinished = f
}

func awaitState(c chan<- api.UpstreamState, stater StateGetter) {
	c <- api.UpstreamState{
		NodeState: cryptkit.NewDigest(longbits.NewBits512FromBytes(stater.State()), SHA3512Digest).AsDigestHolder(),
	}
}
