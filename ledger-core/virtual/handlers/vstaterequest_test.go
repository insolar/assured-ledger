package handlers

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/shareddata"
	"github.com/insolar/assured-ledger/ledger-core/testutils/slotdebugger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object/preservedstatereport"
)

func TestVStateRequest_ProcessObjectWithoutState(t *testing.T) {
	defer commontestutils.LeakTester(t)

	var (
		mc              = minimock.NewController(t)
		pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		smGlobalRef     = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		sharedStateData = smachine.NewUnboundSharedData(&rms.VStateReport{
			Status:              rms.StateStatusEmpty,
			AsOf:                pulse.Unknown,
			Object:              rms.NewReference(smGlobalRef),
			OrderedPendingCount: 1,
			ProvidedContent:     nil,
		})
		smObjectAccessor = preservedstatereport.SharedReportAccessor{SharedDataLink: sharedStateData}
	)

	smVStateRequest := SMVStateRequest{
		Payload: &rms.VStateRequest{
			Object:           rms.NewReference(smGlobalRef),
			RequestedContent: rms.RequestLatestDirtyState,
		},
		reportAccessor: smObjectAccessor,
	}

	execCtx := smachine.NewExecutionContextMock(mc).
		UseSharedMock.Set(shareddata.CallSharedDataAccessor).
		JumpMock.Return(smachine.StateUpdate{})

	smVStateRequest.stepBuildStateReport(execCtx)

	require.True(t, smVStateRequest.objectStateReport.LatestDirtyState.IsZero())
	require.Equal(t, int32(0), smVStateRequest.objectStateReport.UnorderedPendingCount)
	require.Equal(t, int32(1), smVStateRequest.objectStateReport.OrderedPendingCount)
	require.Nil(t, smVStateRequest.objectStateReport.ProvidedContent)

	mc.Finish()
}

func TestDSMVStateRequest_PresentPulse(t *testing.T) {
	defer commontestutils.LeakTester(t)

	var (
		mc  = minimock.NewController(t)
		ctx = instestlogger.TestContext(t)

		objectRef = gen.UniqueGlobalRef()
		caller    = gen.UniqueGlobalRef()

		catalogWrapper                = object.NewCatalogMockWrapper(mc)
		catalog        object.Catalog = catalogWrapper.Mock()
	)

	slotMachine := slotdebugger.New(ctx, t)
	slotMachine.InitEmptyMessageSender(mc)
	slotMachine.AddInterfaceDependency(&catalog)

	smStateRequest := SMVStateRequest{
		Payload: &rms.VStateRequest{
			Object: rms.NewReference(objectRef),
		},
		Meta: &rms.Meta{
			Sender: rms.NewReference(caller),
		},
	}

	slotMachine.Start()
	defer slotMachine.Stop()

	smWrapper := slotMachine.AddStateMachine(ctx, &smStateRequest)

	slotMachine.RunTil(smWrapper.AfterStep(smStateRequest.stepWait))

	slotMachine.Migrate()

	slotMachine.RunTil(smWrapper.BeforeStep(smStateRequest.stepCheckCatalog))

	mc.Finish()
}

func TestDSMVStateRequest_PastPulse(t *testing.T) {
	defer commontestutils.LeakTester(t)

	var (
		mc  = minimock.NewController(t)
		ctx = instestlogger.TestContext(t)

		objectRef = gen.UniqueGlobalRef()
		caller    = gen.UniqueGlobalRef()

		catalogWrapper                = object.NewCatalogMockWrapper(mc)
		catalog        object.Catalog = catalogWrapper.Mock()
	)

	slotMachine := slotdebugger.NewPast(ctx, t)
	slotMachine.InitEmptyMessageSender(mc)
	slotMachine.AddInterfaceDependency(&catalog)

	smStateRequest := SMVStateRequest{
		Payload: &rms.VStateRequest{
			Object: rms.NewReference(objectRef),
		},
		Meta: &rms.Meta{
			Sender: rms.NewReference(caller),
		},
	}

	slotMachine.Start()
	defer slotMachine.Stop()

	smWrapper := slotMachine.AddStateMachine(ctx, &smStateRequest)

	slotMachine.RunTil(smWrapper.BeforeStep(smStateRequest.stepCheckCatalog))

	mc.Finish()
}
