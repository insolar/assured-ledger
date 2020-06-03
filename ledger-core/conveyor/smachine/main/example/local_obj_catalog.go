// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package example

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

type LocalObjectCatalog struct {
}

func (p LocalObjectCatalog) Get(ctx smachine.ExecutionContext, key longbits.ByteString) SharedObjectStateAccessor {
	if v, ok := p.TryGet(ctx, key); ok {
		return v
	}
	panic(fmt.Sprintf("missing entry: %s", key))
}

func (p LocalObjectCatalog) TryGet(ctx smachine.ExecutionContext, key longbits.ByteString) (SharedObjectStateAccessor, bool) {

	if v := ctx.GetPublishedLink(key); v.IsAssignableTo((*SharedObjectState)(nil)) {
		return SharedObjectStateAccessor{v}, true
	}
	return SharedObjectStateAccessor{}, false
}

func (p LocalObjectCatalog) GetOrCreate(ctx smachine.ExecutionContext, key longbits.ByteString) SharedObjectStateAccessor {
	if v, ok := p.TryGet(ctx, key); ok {
		return v
	}

	ctx.InitChild(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return NewVMObjectSM(key)
	})

	return p.Get(ctx, key)
}

////////////////////////////////////////

type SharedObjectStateAccessor struct {
	smachine.SharedDataLink
}

func (v SharedObjectStateAccessor) Prepare(fn func(*SharedObjectState)) smachine.SharedDataAccessor {
	return v.PrepareAccess(func(data interface{}) bool {
		fn(data.(*SharedObjectState))
		return false
	})
}
