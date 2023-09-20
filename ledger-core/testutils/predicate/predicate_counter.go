package predicate

import (
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
)

func NewCounter(predicateFn Func, initCount int) *Counter {
	c := &Counter{predicateFn: predicateFn}
	c.count.Store(initCount)
	return c
}

type Counter struct {
	predicateFn Func
	count       atomickit.Int
}

func (p *Counter) Count() int {
	return p.count.Load()
}

func (p *Counter) EventInput(event debuglogger.UpdateEvent) SubscriberState {
	if p.predicateFn == nil || p.predicateFn(event) {
		p.count.Add(-1)
	}
	return RetainSubscriber
}

func (p *Counter) AfterPositiveToZero(event debuglogger.UpdateEvent) bool {
	prev := p.Count()
	p.EventInput(event)
	return prev > 0 && p.Count() == 0
}

// func (p *Counter) AfterZeroCount(event debuglogger.UpdateEvent) bool {
// 	p.EventInput(event)
// 	return p.Count() == 0
// }
