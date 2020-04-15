// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sm_object // nolint:golint

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

type LocalObjectCatalog struct{}

func (p LocalObjectCatalog) Get(ctx smachine.ExecutionContext, objectReference insolar.Reference) SharedObjectStateAccessor {
	if v, ok := p.TryGet(ctx, objectReference); ok {
		return v
	}
	panic(fmt.Sprintf("missing entry: %s", objectReference.String()))
}

func (p LocalObjectCatalog) TryGet(ctx smachine.ExecutionContext, objectReference insolar.Reference) (SharedObjectStateAccessor, bool) {
	if v := ctx.GetPublishedLink(objectReference); v.IsAssignableTo((*SharedObjectState)(nil)) {
		return SharedObjectStateAccessor{v}, true
	}
	return SharedObjectStateAccessor{}, false
}

func (p LocalObjectCatalog) Create(ctx smachine.ExecutionContext, objectReference insolar.Reference) SharedObjectStateAccessor {
	if _, ok := p.TryGet(ctx, objectReference); ok {
		panic(fmt.Sprintf("already exists: %s", objectReference.String()))
	}

	ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		ctx.SetTracerID(fmt.Sprintf("object-%s", objectReference.String()))
		return NewStateMachineObject(objectReference, false)
	})

	return p.Get(ctx, objectReference)
}

func (p LocalObjectCatalog) GetOrCreate(ctx smachine.ExecutionContext, objectReference insolar.Reference) SharedObjectStateAccessor {
	if v, ok := p.TryGet(ctx, objectReference); ok {
		return v
	}

	ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		ctx.SetTracerID(fmt.Sprintf("object-%s", objectReference.String()))
		return NewStateMachineObject(objectReference, true)
	})

	return p.Get(ctx, objectReference)
}

// //////////////////////////////////////

type SharedObjectStateAccessor struct {
	smachine.SharedDataLink
}

func (v SharedObjectStateAccessor) Prepare(fn func(*SharedObjectState)) smachine.SharedDataAccessor {
	return v.PrepareAccess(func(data interface{}) bool {
		fn(data.(*SharedObjectState))
		return false
	})
}
