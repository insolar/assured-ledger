// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
)

type protoStarter struct {
	ctl   *Controller
	state atomickit.StartStopFlag
	peers uniproto.PeerManager
}

func (p *protoStarter) Start(peers uniproto.PeerManager) {
	if peers == nil {
		panic(throw.IllegalValue())
	}
	if !p.state.DoStart(func() {
		p.peers = peers
		p.ctl.onStarted()
	}) {
		panic(throw.IllegalState())
	}
}

func (p *protoStarter) Stop() {
	if p.state.DoDiscard(nil, nil) {
		p.ctl.onStopped()
	}
}

func (p *protoStarter) NextPulse(pr pulse.Range) {
	if !p.state.IsActive() {
		return
	}
	_, cycle := p.ctl.nextPulseCycle(pr.RightBoundData().PulseNumber)
	go p.ctl.sender.tracks.nextTTLCycle(cycle)
}

func (p *protoStarter) isActive() bool {
	return p.state.IsActive()
}

func (p *protoStarter) wasStarted() bool {
	return p.state.WasStarted()
}
