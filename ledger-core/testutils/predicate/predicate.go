// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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

func Sequence(predicates ...Func) Func {
	pos := 0

	return func(event debuglogger.UpdateEvent) bool {
		if pos == len(predicates) {
			return true
		}

		if predicates[pos](event) == true {
			pos++

			if pos == len(predicates) {
				return true
			}
		}
		return false
	}
}
