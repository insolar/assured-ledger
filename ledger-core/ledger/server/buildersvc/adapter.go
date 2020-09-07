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
	"github.com/insolar/assured-ledger/ledger-core/crypto"
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jetalloc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/dropstorage"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewWriteAdapter(adapterID smachine.AdapterID, executor smachine.AdapterExecutor, service Service) WriteAdapter {
	if service == nil {
		panic(throw.IllegalValue())
	}

	return WriteAdapter{
		adapter: smachine.NewExecutionAdapter(adapterID, executor),
		service: service,
	}
}

func NewReadAdapter(adapterID smachine.AdapterID, executor smachine.AdapterExecutor, service ReadService) ReadAdapter {
	if service == nil {
		panic(throw.IllegalValue())
	}

	return ReadAdapter{
		adapter: smachine.NewExecutionAdapter(adapterID, executor),
		service: service,
	}
}

func NewAdapterComponent(cfg smadapter.Config, ps crypto.PlatformScheme) managed.Component {

	svc := newService(
		jetalloc.NewMaterialAllocationStrategy(false),
		ps.ConsensusScheme().NewMerkleDigester(),

		func(pulse.Number) bundle.SnapshotWriter {
			return dropstorage.NewMemoryStorageWriter(ledger.DefaultDustSection, 1<<17)
		},
	)

	var writeAdapter WriteAdapter
	var readAdapter ReadAdapter

	executors, component := smadapter.NewComponentExt(context.Background(), svc, func(holder managed.Holder) {
		holder.AddDependency(writeAdapter)
		holder.AddDependency(readAdapter)
	}, cfg, cfg)
	writeAdapter = NewWriteAdapter("NewDropBuilder", executors[0], svc)
	readAdapter = NewReadAdapter("NewDropReader", executors[1], svc)
	return component
}

type WriteAdapter struct {
	adapter smachine.ExecutionAdapter
	service Service
}

func (v WriteAdapter) PrepareAsync(ctx smachine.ExecutionContext, callFn func(Service) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
	if callFn == nil {
		panic(throw.IllegalValue())
	}

	return v.adapter.PrepareAsync(ctx, func(context.Context, interface{}) smachine.AsyncResultFunc {
		return callFn(v.service)
	})
}

func (v WriteAdapter) PrepareNotify(ctx smachine.ExecutionContext, callFn func(Service)) smachine.NotifyRequester {
	if callFn == nil {
		panic(throw.IllegalValue())
	}

	return v.adapter.PrepareNotify(ctx, func(context.Context, interface{}) {
		callFn(v.service)
	})
}

/****************************/

type ReadAdapter struct {
	adapter smachine.ExecutionAdapter
	service ReadService
}

func (v ReadAdapter) PrepareAsync(ctx smachine.ExecutionContext, callFn func(ReadService) smachine.AsyncResultFunc) smachine.AsyncCallRequester {
	if callFn == nil {
		panic(throw.IllegalValue())
	}

	return v.adapter.PrepareAsync(ctx, func(context.Context, interface{}) smachine.AsyncResultFunc {
		return callFn(v.service)
	})
}

func (v ReadAdapter) PrepareNotify(ctx smachine.ExecutionContext, callFn func(ReadService)) smachine.NotifyRequester {
	if callFn == nil {
		panic(throw.IllegalValue())
	}

	return v.adapter.PrepareNotify(ctx, func(context.Context, interface{}) {
		callFn(v.service)
	})
}
