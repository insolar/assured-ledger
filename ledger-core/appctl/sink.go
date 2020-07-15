// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package appctl

import (
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type NodeState struct {
	api.UpstreamState
}

type NodeStateChan = chan<- NodeState

func NewNodeStateSink(ch chan NodeState) (NodeStateSink, func(committed bool)) {
	ctl := &sinkCtl{
		report: ch,
	}
	return NodeStateSink{ ctl }, ctl.setReadyState
}

type NodeStateSink struct {
	ctl *sinkCtl
}

func (v NodeStateSink) IsZero() bool {
	return v.ctl == nil
}

func (v NodeStateSink) Occupy() NodeStateChan {
	if !v.ctl.state.CompareAndSetBits(0, sinkStateCommitted, sinkStateOccupied) {
		panic(throw.IllegalState())
	}
	return v.ctl.report
}

func (v NodeStateSink) ReadyChan() synckit.SignalChannel {
	return v.ctl.ready
}

func (v NodeStateSink) IsCommitted() (isReady, isCommitted bool) {
	state := v.ctl.state.Load()
	return state&sinkStateReady != 0, state&sinkStateCommitted != 0
}

const (
	sinkStateReady = 1<<iota
	sinkStateCommitted
	sinkStateOccupied
)

type sinkCtl struct {
	ready synckit.ClosableSignalChannel
	state atomickit.Uint32
	report chan NodeState
}

func (p *sinkCtl) setReadyState(committed bool) {
	state := uint32(sinkStateReady)
	if committed {
		state |= sinkStateCommitted
	}
	if !p.state.TrySetBits(state, true) {
		panic(throw.IllegalState())
	}
	close(p.ready)
}
