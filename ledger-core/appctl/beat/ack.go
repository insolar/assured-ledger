// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package beat

import (
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type AckData struct {
	api.UpstreamState
}

type AckChan = chan<- AckData

func NewAck(ch chan AckData) (Ack, func(ack bool)) {
	sink := &ackSink{
		ready: make(synckit.ClosableSignalChannel),
		report: ch,
	}
	return Ack{sink}, sink.setAck
}

type Ack struct {
	ctl *ackSink
}

func (v Ack) IsZero() bool {
	return v.ctl == nil
}

func (v Ack) Acquire() AckChan {
	if !v.ctl.state.CompareAndSetBits(0, sinkStateAck, sinkStateAcquired) {
		panic(throw.IllegalState())
	}
	return v.ctl.report
}

func (v Ack) DoneChan() synckit.SignalChannel {
	return v.ctl.ready
}

func (v Ack) IsDone() (isDone, isAck bool) {
	state := v.ctl.state.Load()
	return state&sinkStateDone != 0, state&sinkStateAck != 0
}

const (
	sinkStateDone uint32 = 1<<iota
	sinkStateAck
	sinkStateAcquired
)

type ackSink struct {
	ready synckit.ClosableSignalChannel
	state atomickit.Uint32
	report chan AckData
}

func (p *ackSink) setAck(ack bool) {
	state := sinkStateDone
	if ack {
		state |= sinkStateAck
	}
	if !p.state.TrySetBits(state, true) {
		panic(throw.IllegalState())
	}
	close(p.ready)
}
