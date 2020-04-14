// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sm_request // nolint:golint

import (
	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
	"github.com/insolar/assured-ledger/ledger-core/v2/logicrunner/sm_execute_request/outgoing"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
)

type StateMachineSagaAccept struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.SagaCallAcceptNotification

	// sharedStateLink sm_object.SharedObjectStateAccessor
	// externalError   error
}

/* -------- Declaration ------------- */

var declSagaAccept smachine.StateMachineDeclaration = &declarationSagaAccept{}

type declarationSagaAccept struct {
	smachine.StateMachineDeclTemplate
}

func (declarationSagaAccept) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*StateMachineSagaAccept)
	return s.Init
}

func (declarationSagaAccept) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, _ *injector.DependencyInjector) {
	_ = sm.(*StateMachineSagaAccept)
}

/* -------- Instance ------------- */

func (s *StateMachineSagaAccept) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return declSagaAccept
}

func (s *StateMachineSagaAccept) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	return ctx.Jump(s.stepExecuteOutgoing)
}

func (s *StateMachineSagaAccept) stepExecuteOutgoing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// parse outgoing request from virtual record
	virtual := record.Virtual{}
	err := virtual.Unmarshal(s.Payload.Request)
	if err != nil {
		return ctx.Error(err)
	}
	rec := record.Unwrap(&virtual)
	outgoingRequest, ok := rec.(*record.OutgoingRequest)
	if !ok {
		return ctx.Error(errors.Errorf("unexpected request received %T", rec))
	}

	return ctx.Replace(func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &outgoing.ExecuteOutgoingSagaRequest{
			OutgoingRequestReference: *insolar.NewReference(s.Payload.DetachedRequestID),
			RequestObjectReference:   *insolar.NewReference(s.Payload.ObjectID),
			Request:                  outgoingRequest,
		}
	})
}
