// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handlers

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils/shareddata"
)

func TestVStateRequest_ProcessObjectWithoutState(t *testing.T) {
	var (
		mc               = minimock.NewController(t)
		pd               = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		smObjectID       = gen.UniqueIDWithPulse(pd.PulseNumber)
		smGlobalRef      = reference.NewSelf(smObjectID)
		smObject         = object.NewStateMachineObject(smGlobalRef)
		sharedStateData  = smachine.NewUnboundSharedData(&smObject.SharedState)
		smObjectAccessor = object.SharedStateAccessor{SharedDataLink: sharedStateData}
	)

	smObject.SetState(object.Empty)
	smObject.IncrementPotentialPendingCounter(contract.ConstructorIsolation())

	smVStateRequest := SMVStateRequest{
		Payload: &payload.VStateRequest{
			Callee: smGlobalRef,
		},
		stateAccessor: smObjectAccessor,
	}

	execCtx := smachine.NewExecutionContextMock(mc).
		UseSharedMock.Set(shareddata.CallSharedDataAccessor).
		JumpMock.Return(smachine.StateUpdate{})

	smVStateRequest.stepBuildStateReport(execCtx)

	require.True(t, smVStateRequest.objectStateReport.LatestDirtyState.IsZero())
	require.Equal(t, int32(0), smVStateRequest.objectStateReport.UnorderedPendingCount)
	require.Equal(t, int32(1), smVStateRequest.objectStateReport.OrderedPendingCount)
	require.Nil(t, smVStateRequest.objectStateReport.ProvidedContent.LatestDirtyState)

	mc.Finish()
}
