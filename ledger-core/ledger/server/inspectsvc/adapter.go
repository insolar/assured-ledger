// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package inspectsvc

import (
	"context"

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
	return NewAdapterExt("RecordInspector", executor, NewService())
}

type Adapter struct {
	adapter smachine.ExecutionAdapter
	service Service
}

func (v Adapter) PrepareInspectRecordSet(ctx smachine.ExecutionContext, set RegisterRequestSet,
	callbackFn func(smachine.AsyncResultContext, InspectedRecordSet, error)) smachine.AsyncCallRequester {

	return v.adapter.PrepareAsync(ctx, func(context.Context, interface{}) smachine.AsyncResultFunc {
		rs, err := v.service.InspectRecordSet(set)
		if callbackFn == nil {
			panic(err)
		}
		return func(ctx smachine.AsyncResultContext) {
			callbackFn(ctx, rs, err)
		}
	})
}
