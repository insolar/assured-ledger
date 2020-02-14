// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sm_execute_request

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

type RequestCatalog struct{}

func (p RequestCatalog) Get(ctx smachine.ExecutionContext, requestReference insolar.Reference) SharedRequestStateAccessor {
	if v, ok := p.TryGet(ctx, requestReference); ok {
		return v
	}
	panic(fmt.Sprintf("missing entry: %s", requestReference.String()))
}

func (p RequestCatalog) TryGet(ctx smachine.ExecutionContext, requestReference insolar.Reference) (SharedRequestStateAccessor, bool) {
	if v := ctx.GetPublishedLink(requestReference); v.IsAssignableTo((*SharedRequestState)(nil)) {
		return SharedRequestStateAccessor{v}, true
	}
	return SharedRequestStateAccessor{}, false
}

// //////////////////////////////////////

type SharedRequestStateAccessor struct {
	smachine.SharedDataLink
}

func (v SharedRequestStateAccessor) Prepare(fn func(*SharedRequestState)) smachine.SharedDataAccessor {
	return v.PrepareAccess(func(data interface{}) bool {
		fn(data.(*SharedRequestState))
		return false
	})
}
