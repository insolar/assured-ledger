// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package example

import (
	"context"
	"strconv"
	"sync/atomic"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smadapter"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

type GameChooseService interface {
	ChooseGame() GameFactoryFunc
}

type GameChooseAdapter struct {
	exec smachine.ExecutionAdapter
}

func (a *GameChooseAdapter) PrepareAsync(ctx smachine.ExecutionContext, fn func(svc GameChooseService) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(_ context.Context, i interface{}) smachine.AsyncResultFunc {
		return fn(i.(GameChooseService))
	})
}

func NewGameAdapter(ctx context.Context, svc GameChooseService) *GameChooseAdapter {
	exec, ch := smadapter.NewCallChannelExecutor(ctx, -1, false, 1)
	smachine.StartChannelWorkerParallelCalls(ctx, 1, ch, svc)

	return &GameChooseAdapter{smachine.NewExecutionAdapter("GameAdapter", exec)}
}

func NewGameChooseService() GameChooseService {
	return gameChooser{}
}

var _ GameChooseService = gameChooser{}

type gameChooser struct{}

var gameIDCount uint64 // atomic

func (gameChooser) ChooseGame() GameFactoryFunc {
	gameID := strconv.FormatUint(atomic.AddUint64(&gameIDCount, 1)+1, 10)
	return func(players GamePlayers) GameStateMachine {
		return NewGameOfRandom(longbits.WrapStr(gameID), players)
	}
}
