// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package predicate

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
)

func NewAsyncCounter(adapter smachine.AdapterID) *AsyncCounter {
	return &AsyncCounter{adapter: adapter}
}

func NewAnyAsyncCounter() *AsyncCounter {
	return &AsyncCounter{}
}

type AsyncCounter struct {
	adapter smachine.AdapterID
	count atomickit.Int
}

func (p *AsyncCounter) Count() int {
	return p.count.Load()
}

func (p *AsyncCounter) EventInput(event debuglogger.UpdateEvent) {
	switch {
	case event.AdapterID == "":
		return
	case p.adapter == "":
		//
	case p.adapter != event.AdapterID:
		return
	}

	switch event.Data.Flags.AdapterFlags() {
	case smachine.StepLoggerAdapterAsyncCancel, smachine.StepLoggerAdapterAsyncExpiredCancel:
		fallthrough
	case smachine.StepLoggerAdapterAsyncResult, smachine.StepLoggerAdapterAsyncExpiredResult:
		p.count.Add(-1)
	case smachine.StepLoggerAdapterAsyncCall:
		p.count.Add(1)
	}
}

func (p *AsyncCounter) AfterZeroCount(event debuglogger.UpdateEvent) bool {
	p.EventInput(event)
	return p.Count() == 0
}
