/*
 * Copyright 2020 Insolar Network Ltd.
 * All rights reserved.
 * This material is licensed under the Insolar License version 1.0,
 * available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package example

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type SharedPairData struct {
	players [2]smachine.SlotLink
	game    GameFactoryFunc
}

type SharedPairDataLink struct {
	sharedData smachine.SharedDataLink
}

func (v SharedPairDataLink) PrepareAccess(fn func(*SharedPairData) (wakeup bool)) smachine.SharedDataAccessor {
	if fn == nil {
		panic(throw.IllegalValue())
	}
	return v.sharedData.PrepareAccess(func(i interface{}) (wakeup bool) {
		return fn(i.(*SharedPairData))
	})
}
