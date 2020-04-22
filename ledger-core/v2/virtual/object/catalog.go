// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type Catalog struct{}

func (p Catalog) Get(_ smachine.ExecutionContext, _ insolar.Reference) SharedStateAccessor {
	panic(throw.NotImplemented())
}

func (p Catalog) TryGet(ctx smachine.ExecutionContext, objectReference insolar.Reference) (SharedStateAccessor, bool) { // nolintcontractrequester/contractrequester.go:342
	if v := ctx.GetPublishedLink(objectReference.String()); v.IsAssignableTo((*SharedState)(nil)) {
		return SharedStateAccessor{v}, true
	}
	return SharedStateAccessor{}, false
}

func (p Catalog) Create(ctx smachine.ExecutionContext, objectReference insolar.Reference) SharedStateAccessor {
	if _, ok := p.TryGet(ctx, objectReference); ok {
		panic(fmt.Sprintf("already exists: %s", objectReference.String()))
	}

	ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		ctx.SetTracerID(fmt.Sprintf("object-%s", objectReference.String()))
		return NewStateMachineObject(objectReference, false)
	})

	accessor, _ := p.TryGet(ctx, objectReference)
	return accessor
}

func (p Catalog) GetOrCreate(ctx smachine.ExecutionContext, objectReference insolar.Reference) SharedStateAccessor {
	if v, ok := p.TryGet(ctx, objectReference); ok {
		return v
	}

	ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		ctx.SetTracerID(fmt.Sprintf("object-%s", objectReference.String()))
		return NewStateMachineObject(objectReference, true)
	})

	accessor, _ := p.TryGet(ctx, objectReference)
	return accessor
}

// //////////////////////////////////////

type SharedStateAccessor struct {
	smachine.SharedDataLink
}

func (v SharedStateAccessor) Prepare(fn func(*SharedState)) smachine.SharedDataAccessor {
	return v.PrepareAccess(func(data interface{}) bool {
		fn(data.(*SharedState))
		return false
	})
}
