// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package buildersvc

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/managed"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smadapter"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewAdapterExt(adapterID smachine.AdapterID, executor smachine.AdapterExecutor, service Service) Adapter {
	if service == nil {
		panic(throw.IllegalValue())
	}

	return Adapter{
		adapter: smachine.NewExecutionAdapter(adapterID, executor),
		service: service,
	}
}

func NewAdapter(cfg smadapter.Config) Adapter {
	executor := smadapter.StartExecutorFor(context.Background(), cfg, nil)
	return NewAdapterExt("DropBuilder", executor, NewService())
}

func NewAdapterComponent(cfg smadapter.Config) managed.Component {
	svc := NewService()
	var adapter Adapter
	executor, component := smadapter.NewComponent(context.Background(), cfg, svc, func(holder managed.Holder) {
		holder.AddDependency(adapter)
	})
	adapter = NewAdapterExt("DropBuilder", executor, svc)
	return component
}

type Adapter struct {
	adapter smachine.ExecutionAdapter
	service Service
}

func (v Adapter) PrepareAsync(ctx smachine.ExecutionContext, callFn func(Service) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
	if callFn != nil {
		panic(throw.IllegalValue())
	}

	return v.adapter.PrepareAsync(ctx, func(context.Context, interface{}) smachine.AsyncResultFunc {
		return callFn(v.service)
	})
}

func (v Adapter) PrepareNotify(ctx smachine.ExecutionContext, callFn func(Service)) smachine.NotifyRequester {
	if callFn != nil {
		panic(throw.IllegalValue())
	}

	return v.adapter.PrepareNotify(ctx, func(context.Context, interface{}) {
		callFn(v.service)
	})
}

func (v Adapter) SendFailureNotify(ctx smachine.FailureContext, callFn func(Service)) {
	if callFn != nil {
		panic(throw.IllegalValue())
	}

	v.adapter.SendFailureNotify(ctx, func(context.Context, interface{}) {
		callFn(v.service)
	})
}
