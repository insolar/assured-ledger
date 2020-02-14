// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

func NewExecutionAdapter(adapterID AdapterId, executor AdapterExecutor) ExecutionAdapter {
	if adapterID.IsEmpty() {
		panic("illegal value")
	}
	if executor == nil {
		panic("illegal value")
	}
	return ExecutionAdapter{adapterID, executor}
}

type ExecutionAdapter struct {
	adapterID AdapterId
	executor  AdapterExecutor
}

func (p ExecutionAdapter) IsEmpty() bool {
	return p.adapterID.IsEmpty()
}

func (p ExecutionAdapter) GetAdapterID() AdapterId {
	return p.adapterID
}

func (p ExecutionAdapter) PrepareSync(ctx ExecutionContext, fn AdapterCallFunc) SyncCallRequester {
	ec := ctx.(*executionContext)
	return &adapterSyncCallRequest{
		adapterCallRequest{ctx: ec, fn: fn, adapterId: p.adapterID, isLogging: ec.s.getAdapterLogging(),
			executor: p.executor, mode: adapterSyncCallContext}}
}

func (p ExecutionAdapter) PrepareAsync(ctx ExecutionContext, fn AdapterCallFunc) AsyncCallRequester {
	ec := ctx.(*executionContext)
	return &adapterCallRequest{ctx: ec, fn: fn, adapterId: p.adapterID, isLogging: ec.s.getAdapterLogging(),
		executor: p.executor, mode: adapterAsyncCallContext, flags: AutoWakeUp}
}

func (p ExecutionAdapter) PrepareNotify(ctx ExecutionContext, fn AdapterNotifyFunc) NotifyRequester {
	ec := ctx.(*executionContext)
	return &adapterNotifyRequest{ctx: ec, fn: fn, adapterId: p.adapterID, isLogging: ec.s.getAdapterLogging(),
		executor: p.executor, mode: adapterAsyncCallContext}
}
