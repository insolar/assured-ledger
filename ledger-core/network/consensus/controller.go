package consensus

import (
	"context"
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/transport"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/capacity"
)

type candidateController interface {
	AddJoinCandidate(candidate transport.FullIntroductionReader) error
}

type Controller interface {
	AddJoinCandidate(candidate profiles.CandidateProfile) error

	Abort()

	ChangePower(level capacity.Level)
	PrepareLeave() <-chan struct{}
	Leave(leaveReason uint32) <-chan struct{}

	RegisterFinishedNotifier(fn network.OnConsensusFinished)

	Chronicles() api.ConsensusChronicles
}

type controller struct {
	consensusController      api.ConsensusController
	controlFeederInterceptor *adapters.ControlFeederInterceptor
	candidateController      candidateController
	chronicles               api.ConsensusChronicles

	mu        *sync.RWMutex
	notifiers []network.OnConsensusFinished
}

func newController(
	controlFeederInterceptor *adapters.ControlFeederInterceptor,
	candidateController candidateController,
	consensusController api.ConsensusController,
	upstream *adapters.UpstreamController,
	chronicles api.ConsensusChronicles,
) *controller {
	controller := &controller{
		controlFeederInterceptor: controlFeederInterceptor,
		consensusController:      consensusController,
		candidateController:      candidateController,
		chronicles:               chronicles,

		mu: &sync.RWMutex{},
	}

	upstream.SetOnFinished(controller.onFinished)

	return controller
}

func (c *controller) AddJoinCandidate(candidate profiles.CandidateProfile) error {
	return c.candidateController.AddJoinCandidate(candidate)
}

func (c *controller) Abort() {
	c.consensusController.Abort()
}

func (c *controller) ChangePower(level capacity.Level) {
	c.controlFeederInterceptor.Feeder().SetRequiredPowerLevel(level)
}

func (c *controller) PrepareLeave() <-chan struct{} {
	return c.controlFeederInterceptor.PrepareLeave()
}

func (c *controller) Leave(leaveReason uint32) <-chan struct{} {
	return c.controlFeederInterceptor.Leave(leaveReason)
}

func (c *controller) RegisterFinishedNotifier(fn network.OnConsensusFinished) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.notifiers = append(c.notifiers, fn)
}

func (c *controller) onFinished(ctx context.Context, report network.Report) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, n := range c.notifiers {
		go n(ctx, report)
	}
}

func (c *controller) Chronicles() api.ConsensusChronicles {
	return c.chronicles
}
