// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package runner

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type RunMode int

const (
	Undefined RunMode = iota

	Start
	Continue
	Abort
)

type RunState interface {
	GetResult() *executionupdate.ContractExecutionStateUpdate
	GetID() call.ID
}

type runState struct {
	state *executionEventSink
	mode  RunMode
}

func (r runState) GetResult() *executionupdate.ContractExecutionStateUpdate {
	return r.state.GetResult()
}

func (r runState) GetID() call.ID {
	return r.state.id
}

type runnerServiceInterceptor struct {
	svc  *DefaultService
	last *runState
}

var _ UnmanagedService = &runnerServiceInterceptor{}

func (p *runnerServiceInterceptor) ExecutionStart(execution execution.Context) RunState {
	if p.last != nil {
		panic(throw.IllegalState())
	}
	p.last = &runState{p.svc.runPrepare(execution), Start}
	return p.last
}

func (p *runnerServiceInterceptor) ExecutionContinue(run RunState, outgoingResult []byte) {
	if p.last != nil {
		panic(throw.IllegalState())
	}
	var ok bool
	p.last, ok = run.(*runState)
	if !ok {
		panic(throw.IllegalValue())
	}
	p.last.state.input <- outgoingResult
	p.last.mode = Continue
}

func (p *runnerServiceInterceptor) ExecutionAbort(run RunState) {
	if p.last != nil {
		panic(throw.IllegalState())
	}
	var ok bool
	p.last, ok = run.(*runState)
	if !ok {
		panic(throw.IllegalValue())
	}
	p.last.mode = Abort
}

func (p *runnerServiceInterceptor) ExecutionClassify(_ execution.Context) (contract.MethodIsolation, error) {
	panic(throw.Impossible())
}

type awaitedRun struct {
	run      *executionEventSink
	resumeFn func()
}

// ================================= Adapter =================================

type serviceAdapter struct {
	svc          *DefaultService
	runExec      smachine.ExecutionAdapter
	parallelExec smachine.ExecutionAdapter
}

type ServiceAdapter interface {
	PrepareExecutionStart(ctx smachine.ExecutionContext, execution execution.Context, fn func(RunState)) smachine.AsyncCallRequester
	PrepareExecutionContinue(ctx smachine.ExecutionContext, state RunState, outgoingResult []byte, fn func()) smachine.AsyncCallRequester
	PrepareExecutionAbort(ctx smachine.ExecutionContext, state RunState, fn func()) smachine.AsyncCallRequester
	PrepareExecutionClassify(ctx smachine.ExecutionContext, execution execution.Context, fn func(contract.MethodIsolation, error)) smachine.AsyncCallRequester
}

func (a *serviceAdapter) PrepareExecutionStart(ctx smachine.ExecutionContext, execution execution.Context, fn func(RunState)) smachine.AsyncCallRequester {
	if fn == nil {
		panic(throw.IllegalValue())
	}

	return a.runExec.PrepareAsync(ctx, func(_ context.Context, arg interface{}) smachine.AsyncResultFunc {
		state := arg.(UnmanagedService).ExecutionStart(execution)
		return func(ctx smachine.AsyncResultContext) { fn(state) }
	})
}

func (a *serviceAdapter) PrepareExecutionContinue(ctx smachine.ExecutionContext, state RunState, outgoingResult []byte, fn func()) smachine.AsyncCallRequester {
	if state == nil {
		panic(throw.IllegalValue())
	}
	if outgoingResult == nil {
		panic(throw.IllegalValue())
	}

	return a.runExec.PrepareAsync(ctx, func(_ context.Context, arg interface{}) smachine.AsyncResultFunc {
		arg.(UnmanagedService).ExecutionContinue(state, outgoingResult)
		return func(ctx smachine.AsyncResultContext) {
			if fn != nil {
				fn()
			}
		}
	})
}

func (a *serviceAdapter) PrepareExecutionAbort(ctx smachine.ExecutionContext, state RunState, fn func()) smachine.AsyncCallRequester {
	if state == nil {
		panic(throw.IllegalValue())
	}

	return a.runExec.PrepareAsync(ctx, func(_ context.Context, arg interface{}) smachine.AsyncResultFunc {
		arg.(UnmanagedService).ExecutionAbort(state)
		return func(ctx smachine.AsyncResultContext) {
			if fn != nil {
				fn()
			}
		}
	})
}

func (a *serviceAdapter) PrepareExecutionClassify(ctx smachine.ExecutionContext, execution execution.Context, fn func(contract.MethodIsolation, error)) smachine.AsyncCallRequester {
	if fn == nil {
		panic(throw.IllegalValue())
	}

	return a.parallelExec.PrepareAsync(ctx, func(_ context.Context, arg interface{}) smachine.AsyncResultFunc {
		classification, err := a.svc.ExecutionClassify(execution)
		return func(ctx smachine.AsyncResultContext) {
			fn(classification, err)
		}
	})
}

func createRunnerAdapter(ctx context.Context, svc *DefaultService) *serviceAdapter {
	parallelReaders := 16

	runAdapterExecutor, runChannel := smachine.NewCallChannelExecutor(ctx, -1, false, parallelReaders)
	newWorker(ctx, svc).Run(runChannel)

	parallelAdapterExecutor, parallelChannel := smachine.NewCallChannelExecutor(ctx, -1, false, parallelReaders)
	smachine.StartChannelWorkerParallelCalls(ctx, 0, parallelChannel, nil)

	return &serviceAdapter{
		svc:          svc,
		runExec:      smachine.NewExecutionAdapter("RunnerServiceAdapter", runAdapterExecutor),
		parallelExec: smachine.NewExecutionAdapter("RunnerServiceAdapterParallel", parallelAdapterExecutor),
	}
}
