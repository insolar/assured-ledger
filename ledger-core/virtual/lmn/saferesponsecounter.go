// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lmn

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type SafeResponseCounter struct {
	count  int
	wakeUp bool
}

func CounterIncrement(ctx smachine.ExecutionContext, link smachine.SharedDataLink, count int) smachine.StateUpdate {
	if count == 0 {
		return smachine.StateUpdate{}
	}

	accessor := link.PrepareAccess(func(i interface{}) (wakeup bool) {
		obj := i.(*SafeResponseCounter)

		switch {
		case obj.count < 0:
			panic(throw.IllegalState())
		default:
			obj.count += count
		}

		return false
	})

	switch accessor.TryUse(ctx).GetDecision() {
	case smachine.Passed, smachine.Impossible:
		return smachine.StateUpdate{}
	case smachine.NotPassed:
		return ctx.Sleep().ThenRepeat()
	default:
		panic(throw.Impossible())
	}
}

func CounterDecrement(ctx smachine.ExecutionContext, link smachine.SharedDataLink) smachine.StateUpdate {
	accessor := link.PrepareAccess(func(i interface{}) (wakeup bool) {
		obj := i.(*SafeResponseCounter)

		switch {
		case obj.count < 0:
			panic(throw.IllegalState())
		default:
			obj.count--
		}

		return obj.wakeUp
	})

	switch accessor.TryUse(ctx).GetDecision() {
	case smachine.Passed, smachine.Impossible:
		return smachine.StateUpdate{}
	case smachine.NotPassed:
		return ctx.Sleep().ThenRepeat()
	default:
		panic(throw.Impossible())
	}

}

func CounterAwaitZero(ctx smachine.ExecutionContext, link smachine.SharedDataLink) smachine.StateUpdate {
	nextState := ctx.Sleep().ThenRepeat()

	accessor := link.PrepareAccess(func(i interface{}) (wakeup bool) {
		counter, ok := i.(*SafeResponseCounter)
		switch {
		case !ok:
			panic(throw.IllegalState())
		case counter.count < 0:
			panic(throw.IllegalState())
		case counter.count == 0:
			nextState = smachine.StateUpdate{}
			counter.wakeUp = false
		default:
			counter.wakeUp = true
		}

		return false
	})

	switch accessor.TryUse(ctx).GetDecision() {
	case smachine.Passed:
		return nextState
	case smachine.Impossible:
		panic(throw.IllegalState())
	case smachine.NotPassed:
		return ctx.Sleep().ThenRepeat()
	default:
		panic(throw.Impossible())
	}
}
