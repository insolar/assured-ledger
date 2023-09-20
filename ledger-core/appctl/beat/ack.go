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

type AckFunc = func(AckData)

func NewAck(fn AckFunc) (Ack, func(ack bool)) {
	sink := &ackSink{
		ready: make(synckit.ClosableSignalChannel),
		report: fn,
	}
	return Ack{sink}, sink.setAck
}

type Ack struct {
	ctl *ackSink
}

func (v Ack) IsZero() bool {
	return v.ctl == nil
}

func (v Ack) Acquire() AckFunc {
	if !v.ctl.state.CompareAndSetBits(0, sinkStateAck, sinkStateAcquired) {
		panic(throw.IllegalState())
	}
	return v.ctl.doReport
}

func (v Ack) IsAcquired() bool {
	state := v.ctl.state.Load()
	return state&sinkStateAcquired != 0
}

func (v Ack) DoneChan() synckit.SignalChannel {
	return v.ctl.ready
}

func (v Ack) IsDone() (isDone, isAck bool) {
	return v.ctl.isDone()
}

const (
	sinkStateDone uint32 = 1<<iota
	sinkStateAck
	sinkStateAcquired
)

type ackSink struct {
	ready synckit.ClosableSignalChannel
	state atomickit.Uint32
	report AckFunc
}

func (p *ackSink) setAck(ack bool) {
	p._setAck(ack)
}

func (p *ackSink) _setAck(ack bool) bool {
	state := sinkStateDone
	if ack {
		state |= sinkStateAck
	}
	if !p.state.TrySetBits(state, true) {
		return false
	}
	close(p.ready)
	return true
}

func (p *ackSink) isDone() (isDone, isAck bool) {
	state := p.state.Load()
	return state&sinkStateDone != 0, state&sinkStateAck != 0
}

func (p *ackSink) doReport(data AckData) {
	if p._setAck(true) {
		p.report(data)
	}
}
