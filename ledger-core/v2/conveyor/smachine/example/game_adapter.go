/*
 * Copyright 2020 Insolar Network Ltd.
 * All rights reserved.
 * This material is licensed under the Insolar License version 1.0,
 * available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package example

import "github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"

type GameChooseService interface {
	ChooseGame() GameFactoryFunc
}

type GameChooseAdapter struct {
	exec smachine.ExecutionAdapter
}

func (a *GameChooseAdapter) PrepareAsync(ctx smachine.ExecutionContext, fn func(svc GameChooseService) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(i interface{}) smachine.AsyncResultFunc {
		return fn(i.(GameChooseService))
	})
}
