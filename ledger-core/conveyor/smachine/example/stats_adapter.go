// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package example

import (
	"context"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
)

type StatsGetService interface {
	GetStats(g* GameRandom) StatsFactoryFunc
}

type StatsGetAdapter struct {
	exec smachine.ExecutionAdapter
}

func (a *StatsGetAdapter) PrepareAsync(ctx smachine.ExecutionContext, fn func(svc StatsGetService) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(_ context.Context, i interface{}) smachine.AsyncResultFunc {
		return fn(i.(StatsGetService))
	})
}

func NewStatsAdapter(ctx context.Context, svc StatsGetService) *StatsGetAdapter {
	exec, ch := smachine.NewCallChannelExecutor(ctx, -1, false, 1)
	smachine.StartChannelWorkerParallelCalls(ctx, 1, ch, svc)

	return &StatsGetAdapter{smachine.NewExecutionAdapter("StatsAdapter", exec)}
}

func NewStatsGetService() StatsGetService {
	return statsGetter{}
}

var _ StatsGetService = statsGetter{}

type statsGetter struct{}

var statsIDCount uint64 // atomic

func (statsGetter) GetStats(g* GameRandom) StatsFactoryFunc {
	return func() StatsStateMachine {
		return NewStatsOfGame()
	}
}
