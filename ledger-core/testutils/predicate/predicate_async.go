package predicate

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewAsyncCounter(adapter smachine.AdapterID) *AsyncCounter {
	return &AsyncCounter{adapter: adapter}
}

func NewAnyAsyncCounter() *AsyncCounter {
	return &AsyncCounter{}
}

type AsyncCounter struct {
	adapter smachine.AdapterID
	count   atomickit.Int
}

func (p *AsyncCounter) Count() int {
	return p.count.Load()
}

func (p *AsyncCounter) EventInput(event debuglogger.UpdateEvent) SubscriberState {
	switch {
	case event.AdapterID == "":
		return RetainSubscriber
	case p.adapter == "":
		//
	case p.adapter != event.AdapterID:
		return RetainSubscriber
	}

	switch event.Data.Flags.AdapterFlags() {
	case smachine.StepLoggerAdapterAsyncCancel, smachine.StepLoggerAdapterAsyncExpiredCancel:
		fallthrough
	case smachine.StepLoggerAdapterAsyncResult, smachine.StepLoggerAdapterAsyncExpiredResult:
		p.count.Add(-1)
	case smachine.StepLoggerAdapterAsyncCall:
		p.count.Add(1)
	}
	return RetainSubscriber
}

func (p *AsyncCounter) AfterPositiveToZero(event debuglogger.UpdateEvent) bool {
	x := p.Count()
	p.EventInput(event)
	return x > 0 && p.Count() == 0
}

func AfterAsyncCall(id smachine.AdapterID) Func {
	return func(event debuglogger.UpdateEvent) bool {
		switch {
		case event.Data.EventType != smachine.StepLoggerAdapterCall:
		case event.AdapterID != id:
		case event.Data.Flags.AdapterFlags() != smachine.StepLoggerAdapterAsyncCall:
		default:
			return true
		}
		return false
	}
}

func AfterResultOfFirstAsyncCall(id smachine.AdapterID) Func {
	hasCall := false
	callID := uint64(0)
	return func(event debuglogger.UpdateEvent) bool {
		switch {
		case event.Data.EventType != smachine.StepLoggerAdapterCall:
		case event.AdapterID != id:
		case !hasCall:
			if event.Data.Flags.AdapterFlags() == smachine.StepLoggerAdapterAsyncCall {
				hasCall = true
				callID = event.CallID
			}
		case callID != event.CallID:
		default:
			switch event.Data.Flags.AdapterFlags() {
			case smachine.StepLoggerAdapterAsyncCall:
				panic(throw.FailHere("duplicate async call id"))
			case smachine.StepLoggerAdapterAsyncCancel:
				panic(throw.FailHere("async call was cancelled"))
			case smachine.StepLoggerAdapterAsyncExpiredCancel, smachine.StepLoggerAdapterAsyncExpiredResult:
				panic(throw.FailHere("async callback has expired"))
			case smachine.StepLoggerAdapterAsyncResult:
				return true
			}
		}
		return false
	}
}
