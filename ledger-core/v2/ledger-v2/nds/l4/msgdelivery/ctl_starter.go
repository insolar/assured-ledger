// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
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

func (p *protoStarter) isActive() bool {
	return p.state.IsActive()
}
