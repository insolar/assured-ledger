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
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils/shareddata"
)

func TestVStateReport_CreateObjectWithoutState(t *testing.T) {
	var (
		mc               = minimock.NewController(t)
		pd               = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		catalog          = object.NewCatalogMockWrapper(mc)
		smObjectID       = gen.UniqueIDWithPulse(pd.PulseNumber)
		smGlobalRef      = reference.NewSelf(smObjectID)
		smObject         = object.NewStateMachineObject(smGlobalRef)
		sharedStateData  = smachine.NewUnboundSharedData(&smObject.SharedState)
		smObjectAccessor = object.SharedStateAccessor{SharedDataLink: sharedStateData}
	)

	catalog.AddObject(smGlobalRef, smObjectAccessor)
	catalog.AllowAccessMode(object.CatalogMockAccessGetOrCreate)

	smVStateReport := SMVStateReport{
		Payload: &payload.VStateReport{
			Callee:                smGlobalRef,
			UnorderedPendingCount: 1,
			OrderedPendingCount:   1,
			ProvidedContent:       &payload.VStateReport_ProvidedContentBody{},
		},
		objectCatalog: catalog.Mock(),
	}

	execCtx := smachine.NewExecutionContextMock(mc).
		UseSharedMock.Set(shareddata.CallSharedDataAccessor).
		StopMock.Return(smachine.StateUpdate{})

	require.Equal(t, object.Unknown, smObject.GetState())
	smVStateReport.stepProcess(execCtx)
	require.Equal(t, object.Empty, smObject.GetState())
	require.Equal(t, uint8(1), smObject.ActiveUnorderedPendingCount)
	require.Equal(t, uint8(1), smObject.ActiveOrderedPendingCount)
	require.Nil(t, smObject.Descriptor())

	require.NoError(t, catalog.CheckDone())
	mc.Finish()
}
