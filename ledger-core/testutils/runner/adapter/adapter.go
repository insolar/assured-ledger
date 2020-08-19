// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapter

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smadapter"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Service interface {
	ExecutionStart(execution execution.Context) runner.RunState
	ExecutionContinue(run runner.RunState, outgoingResult requestresult.OutgoingExecutionResult)
	ExecutionAbort(run runner.RunState)
	ExecutionClassify(execution execution.Context) (contract.MethodIsolation, error)
}

func (a *Imposter) PrepareExecutionStart(ctx smachine.ExecutionContext, execution execution.Context, fn func(runner.RunState)) smachine.AsyncCallRequester {
	if fn == nil {
		panic(throw.IllegalValue())
	}

	return a.exec.PrepareAsync(ctx, func(_ context.Context, arg interface{}) smachine.AsyncResultFunc {
		state := a.mockedService.ExecutionStart(execution)
		return func(ctx smachine.AsyncResultContext) { fn(state) }
	})
}

func (a *Imposter) PrepareExecutionContinue(ctx smachine.ExecutionContext, state runner.RunState, outgoingResult requestresult.OutgoingExecutionResult, fn func()) smachine.AsyncCallRequester {
	if state == nil {
		panic(throw.IllegalValue())
	}

	return a.exec.PrepareAsync(ctx, func(_ context.Context, arg interface{}) smachine.AsyncResultFunc {
		a.mockedService.ExecutionContinue(state, outgoingResult)
		return func(ctx smachine.AsyncResultContext) {
			if fn != nil {
				fn()
			}
		}
	})
}

func (a *Imposter) PrepareExecutionAbort(ctx smachine.ExecutionContext, state runner.RunState) smachine.AsyncCallRequester {
	if state == nil {
		panic(throw.IllegalValue())
	}

	return a.exec.PrepareAsync(ctx, func(_ context.Context, arg interface{}) smachine.AsyncResultFunc {
		a.mockedService.ExecutionAbort(state)
		return nil
	})
}

func (a *Imposter) PrepareExecutionClassify(ctx smachine.ExecutionContext, execution execution.Context, fn func(contract.MethodIsolation, error)) smachine.AsyncCallRequester {
	if fn == nil {
		panic(throw.IllegalValue())
	}

	return a.exec.PrepareAsync(ctx, func(_ context.Context, arg interface{}) smachine.AsyncResultFunc {
		classification, err := a.mockedService.ExecutionClassify(execution)
		return func(ctx smachine.AsyncResultContext) {
			fn(classification, err)
		}
	})
}

type Imposter struct {
	exec          smachine.ExecutionAdapter
	mockedService Service
}

func NewImposter(ctx context.Context, svc Service, parallelReaders int) *Imposter {
	parallelAdapterExecutor, parallelChannel := smadapter.NewCallChannelExecutor(ctx, -1, false, parallelReaders)
	smachine.StartChannelWorkerParallelCalls(ctx, 0, parallelChannel, nil)

	return &Imposter{
		exec:          smachine.NewExecutionAdapter("RunnerServiceAdapterParallel", parallelAdapterExecutor),
		mockedService: svc,
	}
}
