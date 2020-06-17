// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package journal

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type FeedFunc = func(debuglogger.UpdateEvent)

func NewFeeder(underlying smachine.SlotMachineLogger, feedFn FeedFunc) *Feeder {
	if feedFn == nil {
		panic(throw.IllegalState())
	}

	return &Feeder{
		captor:  debuglogger.NewDebugMachineLoggerNoBlock(underlying, 100),
		feedFn: feedFn,
	}
}

var _ smachine.SlotMachineLogger = &Feeder{}
type captor = debuglogger.DebugMachineLogger

type Feeder struct {
	captor
	feedFn FeedFunc
}

func (p *Feeder) Start() {
	// no need to have a limit
	go func() {
		for {
			rv := p.captor.GetEvent()
			p.feedFn(rv) // allow stop to be detected by receiver(s)
			if rv.IsEmpty() {
				return
			}
		}
	}()
}
