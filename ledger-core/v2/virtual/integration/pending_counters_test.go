// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package small

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/sworker"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender"
	messagesenderadapter "github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender/adapter"
	pulse2 "github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/adapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executor"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/statemachine"
)

type FakeSM struct {
	// input arguments
	Meta    *payload.Meta
	Payload *payload.VCallRequest

	PotentialImmutablePendingCount *uint8
	PotentialMutablePendingCount   *uint8

	// internal data
	objectSharedState object.SharedStateAccessor
}

var dFakeSMInstance smachine.StateMachineDeclaration = &dFakeSM{}

type dFakeSM struct {
	smachine.StateMachineDeclTemplate
}

func (*dFakeSM) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, _ *injector.DependencyInjector) {
}

func (*dFakeSM) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*FakeSM)
	return s.Init
}

func (s *FakeSM) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dFakeSMInstance
}

func (s *FakeSM) prepareExecution(_ smachine.InitializationContext) {
}

func (s *FakeSM) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	s.prepareExecution(ctx)

	return ctx.Jump(s.stepUpdatePendingCounters)
}

func (s *FakeSM) stepUpdatePendingCounters(ctx smachine.ExecutionContext) smachine.StateUpdate {
	objectID := reference.NewGlobalSelf(s.Payload.CallOutgoing)
	objectCatalog := object.Catalog{}
	objectSharedState, ok := objectCatalog.TryGet(ctx, objectID)
	if !ok {
		return ctx.Yield().ThenRepeat()
	}
	switch objectSharedState.Prepare(func(state *object.SharedState) {
		*s.PotentialMutablePendingCount = state.PotentialMutablePendingCount
		*s.PotentialImmutablePendingCount = state.PotentialImmutablePendingCount
	}).TryUse(ctx).GetDecision() {
	case smachine.NotPassed:
		return ctx.WaitShared(objectSharedState.SharedDataLink).ThenRepeat()
	case smachine.Impossible:
		ctx.Log().Fatal("failed to get object state: already dead")
	case smachine.Passed:
	default:
		panic(throw.NotImplemented())
	}
	return ctx.Stop()
}

func Test_SlotMachine_Increment_Pending_Counters(t *testing.T) {
	const scanCountLimit = 1e4

	mc := minimock.NewController(t)
	defer mc.Finish()

	ctx := inslogger.TestContext(t)

	machineConfig := smachine.SlotMachineConfig{
		PollingPeriod:     500 * time.Millisecond,
		PollingTruncate:   1 * time.Millisecond,
		SlotPageSize:      1000,
		ScanCountLimit:    100000,
		SlotMachineLogger: statemachine.ConveyorLoggerFactory{},
	}

	// create VCallRequest
	pd := pulse2.NewFirstPulsarData(10, longbits.Bits256{})
	caller := insolar.Reference{}
	prototype := gen.Reference()
	vCallRequest := payload.VCallRequest{
		Polymorph:           uint32(payload.TypeVCallRequest),
		CallType:            payload.CTConstructor,
		CallFlags:           0,
		CallAsOf:            0,
		Caller:              caller,
		Callee:              gen.Reference(),
		CallSiteDeclaration: prototype,
		CallSiteMethod:      "test",
		CallSequence:        0,
		CallReason:          insolar.Reference{},
		RootTX:              insolar.Reference{},
		CallTX:              insolar.Reference{},
		CallRequestFlags:    0,
		KnownCalleeIncoming: insolar.Reference{},
		EntryHeadHash:       nil,
		CallOutgoing:        reference.Local{},
		Arguments:           nil,
	}

	// create primary state machine
	smExecute := execute.SMExecute{
		Payload: &vCallRequest,
		Meta: &payload.Meta{
			Sender: caller,
		},
	}

	var (
		immutablePendingCount uint8
		mutablePendingCount   uint8
	)
	smFakeSM := FakeSM{
		Payload:                        &vCallRequest,
		PotentialImmutablePendingCount: &immutablePendingCount,
		PotentialMutablePendingCount:   &mutablePendingCount,
	}

	executorMock := testutils.NewMachineLogicExecutorMock(mc)
	executorMock.CallConstructorMock.Return(nil, []byte("345"), nil)
	runnerService := runner.NewService()
	require.NoError(t, runnerService.Init())
	manager := executor.NewManager()
	err := manager.RegisterExecutor(insolar.MachineTypeBuiltin, executorMock)
	require.NoError(t, err)
	runnerService.Manager = manager
	cacheMock := descriptor.NewCacheMock(t)
	runnerService.Cache = cacheMock
	cacheMock.ByPrototypeRefMock.Return(
		descriptor.NewPrototypeDescriptor(gen.Reference(), gen.ID(), gen.Reference()),
		descriptor.NewCodeDescriptor(nil, insolar.MachineTypeBuiltin, gen.Reference()),
		nil,
	)
	runnerAdapter := adapter.CreateRunnerServiceAdapter(ctx, runnerService)

	signal := synckit.NewVersionedSignal()
	slotMachine := smachine.NewSlotMachine(machineConfig,
		signal.NextBroadcast,
		signal.NextBroadcast,
		nil)
	slotMachine.AddDependency(runnerAdapter)

	publisherMock := &utils.PublisherMock{}

	publisherMock.Checker = func(topic string, messages ...*message.Message) error {
		assert.Len(t, messages, 1)

		var (
			_      = messages[0].Metadata
			metaPl = messages[0].Payload
		)

		metaPlType, err := payload.UnmarshalType(metaPl)
		assert.NoError(t, err)
		assert.Equal(t, payload.TypeMeta, metaPlType)

		metaPayload, err := payload.Unmarshal(metaPl)
		assert.NoError(t, err)
		assert.IsType(t, &payload.Meta{}, metaPayload)

		callResultPl := metaPayload.(*payload.Meta).Payload
		callResultPlType, err := payload.UnmarshalType(callResultPl)
		assert.NoError(t, err)
		assert.Equal(t, payload.TypeVCallResult, callResultPlType)

		callResultPayload, err := payload.Unmarshal(callResultPl)
		assert.NoError(t, err)
		assert.IsType(t, &payload.VCallResult{}, callResultPayload)
		assert.Equal(t, callResultPayload.(*payload.VCallResult).ReturnArguments, []byte("345"))

		return nil
	}

	jetCoordinatorMock := jet.NewCoordinatorMock(t).
		MeMock.Return(gen.Reference()).
		QueryRoleMock.Return([]insolar.Reference{gen.Reference()}, nil)
	pulses := pulse.NewStorageMem()
	messageSender := messagesender.NewDefaultService(publisherMock, jetCoordinatorMock, pulses)
	messageSenderAdapter := messagesenderadapter.CreateMessageSendService(ctx, messageSender)
	slotMachine.AddDependency(messageSenderAdapter)

	pulseSlot := conveyor.NewPresentPulseSlot(nil, pd.AsRange())
	slotMachine.AddDependency(&pulseSlot)

	slotMachine.AddNewByFunc(ctx, func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &smExecute
	}, smachine.CreateDefaultValues{})
	slotMachine.AddNewByFunc(ctx, func(ctx smachine.ConstructionContext) smachine.StateMachine {
		return &smFakeSM
	}, smachine.CreateDefaultValues{})

	workerFactory := sworker.NewAttachableSimpleSlotWorker()
	neverSignal := synckit.NewNeverSignal()
	require.Equal(t, uint8(0), mutablePendingCount)
	require.Equal(t, uint8(0), immutablePendingCount)
	for {
		var (
			repeatNow    bool
			nextPollTime time.Time
		)
		wakeupSignal := signal.Mark()
		workerFactory.AttachTo(slotMachine, neverSignal, scanCountLimit, func(worker smachine.AttachedSlotWorker) {
			repeatNow, nextPollTime = slotMachine.ScanOnce(0, worker)
		})
		if slotMachine.OccupiedSlotCount() < 2 {
			break
		}
		switch {
		case repeatNow:
			continue
		case !nextPollTime.IsZero():
			time.Sleep(time.Until(nextPollTime))
		case !slotMachine.IsActive():
			break
		default:
			wakeupSignal.Wait()
		}
	}
	require.Equal(t, uint8(1), mutablePendingCount)
	require.Equal(t, uint8(0), immutablePendingCount)
}
