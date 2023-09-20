package predicate

import (
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
)

type Func = func(debuglogger.UpdateEvent) bool

func Never() Func {
	return func(debuglogger.UpdateEvent) bool {
		return false
	}
}

func Ever() Func {
	return func(debuglogger.UpdateEvent) bool {
		return true
	}
}

func Not(predicate Func) Func {
	return func(event debuglogger.UpdateEvent) bool {
		return !predicate(event)
	}
}

func And(predicates ...Func) Func {
	if len(predicates) == 0 {
		return Never()
	}
	return func(event debuglogger.UpdateEvent) bool {
		for _, fn := range predicates {
			if !fn(event) {
				return false
			}
		}
		return true
	}
}

func Or(predicates ...Func) Func {
	return func(event debuglogger.UpdateEvent) bool {
		for _, fn := range predicates {
			if fn(event) {
				return true
			}
		}
		return false
	}
}

// ChainOf will wait for all predicates to fire on different steps
// for example:
// ChainOf(X1, X2, X3) will wait for:
// * predicate X1 eventually to return true
// * predicate X2 eventually to return true (no more checks of X1 ever)
// * and predicate X3 eventually to return true (no checks of X1 and X2)
// and only then it'll return true
func ChainOf(predicates ...Func) Func {
	pos := 0

	return func(event debuglogger.UpdateEvent) bool {
		if pos == len(predicates) {
			return true
		}

		if predicates[pos](event) {
			pos++

			if pos == len(predicates) {
				return true
			}
		}
		return false
	}
}
