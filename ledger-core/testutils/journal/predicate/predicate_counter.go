// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package predicate

import (
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
)

func NewCounter(predicateFn Func, initCount int) *Counter {
	return &Counter{predicateFn: predicateFn}
}

type Counter struct {
	predicateFn Func
	count       atomickit.Int
}

func (p *Counter) Count() int {
	return p.count.Load()
}

func (p *Counter) EventInput(event debuglogger.UpdateEvent) {
	if p.predicateFn == nil || p.predicateFn(event) {
		p.count.Add(-1)
	}
}

func (p *Counter) AfterZeroCount(event debuglogger.UpdateEvent) bool {
	p.EventInput(event)
	return p.Count() == 0
}
