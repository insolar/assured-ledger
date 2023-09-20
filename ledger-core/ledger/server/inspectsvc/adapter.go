package inspectsvc

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/managed"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smadapter"
	"github.com/insolar/assured-ledger/ledger-core/crypto"
	"github.com/insolar/assured-ledger/ledger-core/reference"
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

func NewAdapterComponent(cfg smadapter.Config, localRef reference.Holder, ps crypto.PlatformScheme) managed.Component {
	svc := NewService(localRef, ps.RecordScheme())

	var adapter Adapter
	executor, component := smadapter.NewComponent(context.Background(), cfg, svc, func(holder managed.Holder) {
		holder.AddDependency(adapter)
	})
	adapter = NewAdapterExt("RecordInspector", executor, svc)
	return component
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
			if err != nil {
				panic(err)
			}
		}
		return func(ctx smachine.AsyncResultContext) {
			callbackFn(ctx, rs, err)
		}
	})
}
