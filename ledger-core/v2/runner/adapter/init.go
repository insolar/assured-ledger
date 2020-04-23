// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapter

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner"
)

type Runner struct {
	svc  runner.Service
	exec smachine.ExecutionAdapter
}

func (a *Runner) PrepareSync(ctx smachine.ExecutionContext, fn func(svc runner.Service)) smachine.SyncCallRequester {
	return a.exec.PrepareSync(ctx, func(_ interface{}) smachine.AsyncResultFunc {
		fn(a.svc)
		return nil
	})
}

func (a *Runner) PrepareAsync(ctx smachine.ExecutionContext, fn func(svc runner.Service) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
	return a.exec.PrepareAsync(ctx, func(_ interface{}) smachine.AsyncResultFunc {
		return fn(a.svc)
	})
}

func (a *Runner) PrepareNotify(
	ctx smachine.ExecutionContext,
	fn func(svc runner.Service),
) smachine.NotifyRequester {
	return a.exec.PrepareNotify(ctx, func(interface{}) {
		fn(a.svc)
	})
}

func CreateRunnerServiceAdapter(ctx context.Context, runnerService runner.Service) *Runner {
	ae, ch := smachine.NewCallChannelExecutor(ctx, -1, false, 16)
	smachine.StartChannelWorkerParallelCalls(ctx, 0, ch, nil)

	return &Runner{
		svc:  runnerService,
		exec: smachine.NewExecutionAdapter("ArtifactClientService", ae),
	}
}
