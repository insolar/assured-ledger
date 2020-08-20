// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package journal

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils/predicate"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func New() *Journal {
	j := &Journal{}
	j.async = predicate.NewAnyAsyncCounter()
	j.dispenser.Subscribe(j.async.EventInput)
	return j
}

type Journal struct {
	dispenser Dispenser
	replay    *Replay
	async     *predicate.AsyncCounter
}

func (p *Journal) InterceptSlotMachineLog(underlying smachine.SlotMachineLogger, stopSignal synckit.SignalChannel) smachine.SlotMachineLogger {
	return NewFeeder(underlying, p.dispenser.EventInput, stopSignal)
}

func (p *Journal) StartRecording(limit int, discardOnOverflow bool) {
	switch {
	case p.replay != nil:
		panic(throw.IllegalState())
	case limit <= 0:
		panic(throw.IllegalValue())
	}
	p.replay = NewReplay(&p.dispenser, limit)
	if discardOnOverflow {
		p.replay.SetDiscardOnOverflow()
	}
	p.dispenser.Subscribe(p.replay.EventInput)
}

func (p *Journal) ReplayAndSubscribe(outFn predicate.SubscriberFunc) {
	if p.replay == nil {
		panic(throw.IllegalState())
	}
	p.replay.ReplayAndSubscribe(outFn)
}

func (p *Journal) TryReplayThenSubscribe(outFn predicate.SubscriberFunc) bool {
	if p.replay == nil {
		p.dispenser.Subscribe(outFn)
		return false
	}
	return p.replay.TryReplayThenSubscribe(outFn)
}

func (p *Journal) Subscribe(outFn predicate.SubscriberFunc) {
	p.dispenser.Subscribe(outFn)
}

func (p *Journal) WaitOnce(pFn predicate.Func) synckit.SignalChannel {
	ch := make(synckit.ClosableSignalChannel, 1)
	p.Subscribe(predicate.EmitOnce(pFn, true, ch))
	return ch
}
func (p *Journal) ReplayAndWaitOnce(pFn predicate.Func) synckit.SignalChannel {
	ch := make(synckit.ClosableSignalChannel, 1)
	p.ReplayAndSubscribe(predicate.EmitOnce(pFn, true, ch))
	return ch
}

func (p *Journal) Wait(pFn predicate.Func) synckit.SignalChannel {
	ch := make(synckit.ClosableSignalChannel, 1)
	p.Subscribe(predicate.EmitByFilter(pFn, true, ch))
	return ch
}

func (p *Journal) ReplayAndWait(pFn predicate.Func) synckit.SignalChannel {
	ch := make(synckit.ClosableSignalChannel, 1)
	p.ReplayAndSubscribe(predicate.EmitByFilter(pFn, true, ch))
	return ch
}

func (p *Journal) ReplayAndWaitStopOf(sample smachine.StateMachine, stopCount int) synckit.SignalChannel {
	return p.ReplayAndWaitOnce(
		predicate.NewCounter(
			predicate.NewSMTypeFilter(sample, predicate.AfterAnyStopOrError),
			stopCount,
		).AfterPositiveToZero)
}

func (p *Journal) WaitStopOf(sample smachine.StateMachine, stopCount int) synckit.SignalChannel {
	return p.WaitOnce(
		predicate.NewCounter(
			predicate.NewSMTypeFilter(sample, predicate.AfterAnyStopOrError),
			stopCount,
		).AfterPositiveToZero)
}

func (p *Journal) WaitInitOf(sample smachine.StateMachine, initCount int) synckit.SignalChannel {
	return p.WaitOnce(
		predicate.NewCounter(
			predicate.NewSMTypeFilter(sample, predicate.AfterInit),
			initCount,
		).AfterPositiveToZero)
}

func (p *Journal) WaitAllAsyncCallsDone() synckit.SignalChannel {
	ch := p.WaitOnce(func(debuglogger.UpdateEvent) bool {
		return p.async.Count() == 0
	})

	if p.async.Count() == 0 {
		return synckit.ClosedChannel()
	}

	return ch
}
