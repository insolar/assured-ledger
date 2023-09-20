package journal

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils/predicate"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type FeedFunc = predicate.SubscriberFunc

func NewFeeder(underlying smachine.SlotMachineLogger, feedFn FeedFunc, stopSignal synckit.SignalChannel) *Feeder {
	if feedFn == nil {
		panic(throw.IllegalState())
	}

	feeder := &Feeder{
		captor: debuglogger.NewDebugMachineLoggerNoBlock(underlying, 100),
		feedFn: feedFn,
	}
	feeder.Start(stopSignal)
	return feeder
}

var _ smachine.SlotMachineLogger = &Feeder{}

type captor = *debuglogger.DebugMachineLogger

type Feeder struct {
	captor
	feedFn FeedFunc
}

func (p *Feeder) Start(stopSignal synckit.SignalChannel) {
	go func() {
		for {
			select {
			case event, ok := <-p.captor.EventChan():
				switch {
				case !ok || event.IsEmpty():
					p.feedFn(event)
					return
				case p.feedFn(event) == predicate.RetainSubscriber:
					continue
				}
			case <-stopSignal:
				//
			}

			p.captor.FlushEvents(synckit.ClosedChannel(), true)
			return
		}
	}()
}
