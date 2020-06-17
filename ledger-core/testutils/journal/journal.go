// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package journal

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func New() *Journal {
	j := &Journal{}
	return j
}

type Journal struct {
	dispenser Dispenser
	replay *Replay
	asyncChan synckit.SignalChannel
}

func (p *Journal) InterceptSlotMachineLog(underlying smachine.SlotMachineLogger) smachine.SlotMachineLogger {
	feeder := NewFeeder(underlying, p.dispenser.EventInput)
	feeder.Start()
	return feeder
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
}

func (p *Journal) ReplayAndSubscribe(dispenser *Dispenser) {
	switch {
	case dispenser == nil:
		panic(throw.IllegalValue())
	case p.replay == nil:
		panic(throw.IllegalState())
	default:
		p.replay.ReplayAndSubscribe(dispenser)
	}
}

func (p *Journal) TryReplayThenSubscribe(dispenser *Dispenser) bool {
	switch {
	case dispenser == nil:
		panic(throw.IllegalValue())
	case p.replay == nil:
		p.dispenser.SubscribeAll(dispenser.EventInput)
		return false
	default:
		return p.replay.TryReplayThenSubscribe(dispenser)
	}
}

func (p *Journal) Subscribe(dispenser *Dispenser) {
	if dispenser == nil {
		panic(throw.IllegalValue())
	}
	p.dispenser.SubscribeAll(dispenser.EventInput)
}
