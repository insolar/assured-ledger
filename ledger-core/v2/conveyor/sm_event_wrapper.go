// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package conveyor

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

type wrapEventSM struct {
	smachine.StateMachineDeclTemplate

	pn       pulse.Number
	ps       *PulseSlot
	createFn smachine.CreateFunc
}

func (sm *wrapEventSM) stepTerminateEvent(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// To properly propagate termination of this SM, we have to get termination of the to-be-SM we are wrapping.
	// So we will run the intended creation procedure and capture its termination handler, but discard SM.

	defResult := ctx.GetDefaultTerminationResult()
	interceptor := &constructionInterceptor{createFn: sm.createFn, defResult: defResult}

	ctx.NewChildExt(interceptor.Create, smachine.CreateDefaultValues{
		Context: ctx.GetContext(),
		Parent:  ctx.ParentLink(),
	})

	if v, ok := defResult.(error); ok {
		return ctx.Error(v)
	}
	return ctx.Stop()
}

type constructionInterceptor struct {
	smachine.ConstructionContext

	createFn  smachine.CreateFunc
	defResult interface{}
}

func (p *constructionInterceptor) Create(ctx smachine.ConstructionContext) smachine.StateMachine {
	if p.ConstructionContext != nil {
		panic("illegal state")
	}
	p.ConstructionContext = ctx
	if p.defResult != nil {
		p.SetDefaultTerminationResult(p.defResult)
	}
	_ = p.createFn(p) // we ignore the created SM

	return nil // stop creation process and trigger termination
}
