package handlers

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/shareddata"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

func TestVStateReport_CreateObjectWithoutState(t *testing.T) {
	defer commontestutils.LeakTester(t)

	var (
		mc               = minimock.NewController(t)
		pd               = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		catalog          = object.NewCatalogMockWrapper(mc)
		smGlobalRef      = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		smObject         = object.NewStateMachineObject(smGlobalRef)
		sharedStateData  = smachine.NewUnboundSharedData(&smObject.SharedState)
		smObjectAccessor = object.SharedStateAccessor{SharedDataLink: sharedStateData}
		pulseSlot        = conveyor.NewPastPulseSlot(nil, pd.AsRange())
	)

	catalog.AddObject(smGlobalRef, smObjectAccessor)
	catalog.AllowAccessMode(object.CatalogMockAccessGetOrCreate)

	smVStateReport := SMVStateReport{
		Payload: &rms.VStateReport{
			Status:                rms.StateStatusEmpty,
			Object:                rms.NewReference(smGlobalRef),
			AsOf:                  pd.PulseNumber,
			UnorderedPendingCount: 1,
			OrderedPendingCount:   1,
			ProvidedContent:       &rms.VStateReport_ProvidedContentBody{},
		},
		objectCatalog: catalog.Mock(),
		pulseSlot:     &pulseSlot,
	}

	execCtx := smachine.NewExecutionContextMock(mc).
		UseSharedMock.Set(shareddata.CallSharedDataAccessor).
		StopMock.Return(smachine.StateUpdate{})

	require.Equal(t, object.Unknown, smObject.GetState())
	smVStateReport.stepProcess(execCtx)
	require.Equal(t, object.Empty, smObject.GetState())
	require.Equal(t, uint8(1), smObject.PreviousExecutorUnorderedPendingCount)
	require.Equal(t, uint8(1), smObject.PreviousExecutorOrderedPendingCount)
	require.Nil(t, smObject.DescriptorDirty())

	require.NoError(t, catalog.CheckDone())
	mc.Finish()
}

func TestVStateReport_StopSMIfAsOfOutdated(t *testing.T) {
	defer commontestutils.LeakTester(t)

	var (
		emptyEntropyFn = func() longbits.Bits256 {
			return longbits.Bits256{}
		}

		mc = minimock.NewController(t)

		pdPMinusThree = pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})
		pdPMinusTwo   = pdPMinusThree.CreateNextPulse(emptyEntropyFn)
		pdPMinusOne   = pdPMinusTwo.CreateNextPulse(emptyEntropyFn)
		pd            = pdPMinusOne.CreateNextPulse(emptyEntropyFn)

		catalog          = object.NewCatalogMockWrapper(mc)
		initState        = []byte("init state")
		smGlobalRef      = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		initRef          = reference.NewRecordOf(smGlobalRef, gen.UniqueLocalRefWithPulse(pd.PulseNumber))
		class            = gen.UniqueGlobalRefWithPulse(pdPMinusThree.PulseNumber)
		smObject         = object.NewStateMachineObject(smGlobalRef)
		sharedStateData  = smachine.NewUnboundSharedData(&smObject.SharedState)
		smObjectAccessor = object.SharedStateAccessor{SharedDataLink: sharedStateData}
	)

	catalog.AddObject(smGlobalRef, smObjectAccessor)
	catalog.AllowAccessMode(object.CatalogMockAccessGetOrCreate)

	pulseSlot := conveyor.NewPresentPulseSlot(nil, pd.AsRange())

	table := []struct {
		name     string
		outdated bool
		asOf     pulse.Number
	}{
		{name: "pulse-1", outdated: false, asOf: pdPMinusOne.PulseNumber},
		{name: "pulse-2", outdated: true, asOf: pdPMinusTwo.PulseNumber},
		{name: "pulse-3", outdated: true, asOf: pdPMinusThree.PulseNumber},
	}

	for _, testCase := range table {
		t.Run(testCase.name, func(t *testing.T) {
			mc := minimock.NewController(t)

			smVStateReport := SMVStateReport{
				Payload: &rms.VStateReport{
					Status: rms.StateStatusReady,
					Object: rms.NewReference(smGlobalRef),
					AsOf:   testCase.asOf,
					ProvidedContent: &rms.VStateReport_ProvidedContentBody{
						LatestDirtyState: &rms.ObjectState{
							Reference: rms.NewReference(initRef),
							Class:     rms.NewReference(class),
							Memory:    rms.NewBytes(initState),
						},
						LatestValidatedState: &rms.ObjectState{
							Reference: rms.NewReference(initRef),
							Class:     rms.NewReference(class),
							Memory:    rms.NewBytes(initState),
						},
					},
				},
				pulseSlot:     &pulseSlot,
				objectCatalog: catalog.Mock(),
			}

			execCtx := smachine.NewExecutionContextMock(mc)

			if testCase.outdated {
				execCtx.JumpMock.Set(commontestutils.AssertJumpStep(t, smVStateReport.stepAsOfOutdated)).
					LogMock.Return(smachine.Logger{})
			} else {
				execCtx.UseSharedMock.Set(shareddata.CallSharedDataAccessor).
					StopMock.Return(smachine.StateUpdate{})
			}

			smVStateReport.stepProcess(execCtx)

			mc.Finish()
		})
	}

}
