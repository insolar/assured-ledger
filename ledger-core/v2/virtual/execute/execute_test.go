// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"testing"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
)

func TestSMExecute_Init_Increase_Pending_Counter(t *testing.T) {
	mc := minimock.NewController(t)
	pulse, _ := insolar.NewPulseNumberFromStr("123456")
	smObjectID := gen.IDWithPulse(pulse)
	smGlobalRef := reference.NewGlobalSelf(smObjectID)
	sm := object.NewStateMachineObject(smGlobalRef, true)
	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			Polymorph:           uint32(payload.TypeVCallRequest),
			CallType:            payload.CTConstructor,
			CallFlags:           nil,
			CallAsOf:            0,
			CallSiteDeclaration: testwallet.GetPrototype(),
			CallSiteMethod:      "New",
			CallRequestFlags:    0,
			CallOutgoing:        smObjectID,
			Arguments:           insolar.MustSerialize([]interface{}{}),
		},
	}
	execCtx := smachine.NewExecutionContextMock(mc)
	execCtx.GetPublishedLinkMock.Expect(smGlobalRef).Return(smachine.NewUnboundSharedData(&sm.SharedState))
	smExecute.stepWaitObjectReady(execCtx)

}
