// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewExecutionAdapter(adapterID AdapterID, executor AdapterExecutor) ExecutionAdapter {
	switch {
	case adapterID.IsEmpty():
		panic(throw.IllegalValue())
	case executor == nil:
		panic(throw.IllegalValue())
	}
	return ExecutionAdapter{adapterID, executor}
}

type ExecutionAdapter struct {
	adapterID AdapterID
	executor  AdapterExecutor
}

func (p ExecutionAdapter) IsEmpty() bool {
	return p.adapterID.IsEmpty()
}

func (p ExecutionAdapter) GetAdapterID() AdapterID {
	return p.adapterID
}

func (p ExecutionAdapter) PrepareSync(ctx ExecutionContext, fn AdapterCallFunc) SyncCallRequester {
	if fn == nil {
		panic(throw.IllegalValue())
	}
	ec := ctx.(*executionContext)
	return &adapterSyncCallRequest{
		adapterCallRequest{ctx: ec, fn: fn, adapterID: p.adapterID, isLogging: ec.s.getAdapterLogging(),
			executor: p.executor, mode: adapterSyncCallContext}}
}

func (p ExecutionAdapter) PrepareAsync(ctx ExecutionContext, fn AdapterCallFunc) AsyncCallRequester {
	if fn == nil {
		panic(throw.IllegalValue())
	}
	ec := ctx.(*executionContext)
	return &adapterCallRequest{ctx: ec, fn: fn, adapterID: p.adapterID, isLogging: ec.s.getAdapterLogging(),
		executor: p.executor, mode: adapterAsyncCallContext, flags: AutoWakeUp}
}

func (p ExecutionAdapter) PrepareNotify(ctx ExecutionContext, fn AdapterNotifyFunc) NotifyRequester {
	if fn == nil {
		panic(throw.IllegalValue())
	}
	ec := ctx.(*executionContext)
	return &adapterNotifyRequest{ctx: ec, fn: fn, adapterID: p.adapterID, isLogging: ec.s.getAdapterLogging(),
		executor: p.executor, mode: adapterAsyncCallContext}
}

func (p ExecutionAdapter) SendFailureNotify(ctx FailureContext, fn AdapterNotifyFunc) {
	if fn == nil {
		panic(throw.IllegalValue())
	}
	fc := ctx.(*failureContext)
	ec := &executionContext{slotContext: fc.slotContext}

	rq := &adapterNotifyRequest{ctx: ec, fn: fn, adapterID: p.adapterID, isLogging: ec.s.getAdapterLogging(),
		executor: p.executor, mode: adapterAsyncCallContext}
	rq.Send()
}
