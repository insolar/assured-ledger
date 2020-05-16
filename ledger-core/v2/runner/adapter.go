// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package runner

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/call"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type RunMode int

const (
	Undefined RunMode = iota

	Start
	Continue
	Abort
)

type RunState struct {
	state *executionEventSink
	mode  RunMode
}

func (r *RunState) GetResult() *executionupdate.ContractExecutionStateUpdate {
	return r.state.GetResult()
}

func (r RunState) GetID() call.ID {
	return r.state.id
}

type runnerServiceInterceptor struct {
	svc  *DefaultService
	last *RunState
}

var _ Service = &runnerServiceInterceptor{}

func (p *runnerServiceInterceptor) ExecutionStart(execution execution.Context) *RunState {
	if p.last != nil {
		panic(throw.IllegalState())
	}
	p.last = &RunState{p.svc.runPrepare(execution), Start}
	return p.last
}

func (p *runnerServiceInterceptor) ExecutionContinue(run *RunState, outgoingResult []byte) {
	if p.last != nil {
		panic(throw.IllegalState())
	}
	p.last = run
	run.state.input <- outgoingResult
	run.mode = Continue
}

func (p *runnerServiceInterceptor) ExecutionAbort(run *RunState) {
	if p.last != nil {
		panic(throw.IllegalState())
	}
	p.last = run
	run.mode = Abort
}

type awaitedRun struct {
	run      *executionEventSink
	resumeFn func()
}

// ================================= Adapter =================================

type ServiceAdapter struct {
	svc    *DefaultService
	exec   smachine.ExecutionAdapter
	worker *worker
}

func (a *ServiceAdapter) PrepareExecutionStart(ctx smachine.ExecutionContext, execution execution.Context, fn func(*RunState)) smachine.AsyncCallRequester {
	if fn == nil {
		panic(throw.IllegalValue())
	}

	return a.exec.PrepareAsync(ctx, func(_ context.Context, arg interface{}) smachine.AsyncResultFunc {
		state := arg.(Service).ExecutionStart(execution)
		return func(ctx smachine.AsyncResultContext) { fn(state) }
	})
}

func (a *ServiceAdapter) PrepareExecutionContinue(ctx smachine.ExecutionContext, state *RunState, outgoingResult []byte, fn func()) smachine.AsyncCallRequester {
	if state == nil {
		panic(throw.IllegalValue())
	}
	if outgoingResult == nil {
		panic(throw.IllegalValue())
	}

	return a.exec.PrepareAsync(ctx, func(_ context.Context, arg interface{}) smachine.AsyncResultFunc {
		arg.(Service).ExecutionContinue(state, outgoingResult)
		return func(ctx smachine.AsyncResultContext) {
			if fn != nil {
				fn()
			}
		}
	})
}

func (a *ServiceAdapter) PrepareExecutionAbort(ctx smachine.ExecutionContext, state *RunState, fn func()) smachine.AsyncCallRequester {
	if state == nil {
		panic(throw.IllegalValue())
	}

	return a.exec.PrepareAsync(ctx, func(_ context.Context, arg interface{}) smachine.AsyncResultFunc {
		arg.(Service).ExecutionAbort(state)
		return func(ctx smachine.AsyncResultContext) {
			if fn != nil {
				fn()
			}
		}
	})
}

func CreateRunnerService(ctx context.Context, svc *DefaultService) *ServiceAdapter {
	parallelReaders := 16
	ae, ch := smachine.NewCallChannelExecutor(ctx, -1, false, parallelReaders)

	return &ServiceAdapter{
		svc:    svc,
		exec:   smachine.NewExecutionAdapter("RunnerServiceAdapter", ae),
		worker: newWorker(ctx, svc).Run(ch),
	}
}
