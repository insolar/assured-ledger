package execute

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	commonTestUtils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/testutils/mocklog"
	"github.com/insolar/assured-ledger/ledger-core/testutils/shareddata"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callregistry"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	memoryCacheAdapter "github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/adapter"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	virtualtestutils "github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/virtualdebugger"
	"github.com/insolar/assured-ledger/ledger-core/virtual/tool"
	"github.com/insolar/assured-ledger/ledger-core/virtual/vnlmn"
)

func executeLeakCheck(t *testing.T) {
	// TODO: remove this ignores after fix closing adapters on conveyor shutdown
	commonTestUtils.LeakTester(t,
		goleak.IgnoreTopFunction("github.com/insolar/assured-ledger/ledger-core/runner.(*worker).Run.func1"),
		goleak.IgnoreTopFunction("github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.startChannelWorkerUnlimParallel.func1"),
	)
}

func expectedInitState(ctx context.Context, sm SMExecute) SMExecute {
	sm.execution.Context = ctx
	sm.execution.Sequence = 0
	sm.execution.Request = sm.Payload
	sm.execution.Pulse = sm.pulseSlot.PulseData()

	if sm.Payload.CallType == rms.CallTypeConstructor {
		sm.isConstructor = true
		sm.execution.Object = reference.NewSelf(sm.Payload.CallOutgoing.GetValue().GetLocal())
	} else {
		sm.execution.Object = sm.Payload.Callee.GetValue()
	}

	sm.execution.Incoming = reference.NewRecordOf(sm.Payload.Callee.GetValue(), sm.Payload.CallOutgoing.GetValue().GetLocal())
	sm.execution.Outgoing = sm.Payload.CallOutgoing.GetValue()

	sm.execution.Isolation = contract.MethodIsolation{
		Interference: sm.Payload.CallFlags.GetInterference(),
		State:        sm.Payload.CallFlags.GetState(),
	}
	sm.lmnContext = vnlmn.NewDummyRegistrationCtx(sm.execution.Object)
	return sm
}

func TestSMExecute_Init(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		ctx = instestlogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd           = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot    = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		caller       = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		callee       = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		callOutgoing = reference.NewRecordOf(caller, gen.UniqueLocalRefWithPulse(pd.PulseNumber))
		meRef        = gen.UniqueGlobalRef()

		callFlags = rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty)

		request = &rms.VCallRequest{
			CallType:       rms.CallTypeConstructor,
			CallFlags:      callFlags,
			CallSiteMethod: "New",
			Caller:         rms.NewReference(caller),
			Callee:         rms.NewReference(callee),
			CallOutgoing:   rms.NewReference(callOutgoing),
			Arguments:      rms.NewBytes(insolar.MustSerialize([]interface{}{})),
		}

		// smGlobalRef = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		objectRef        = virtualtestutils.GetObjectReference(request, meRef)
		smObject         = object.NewStateMachineObject(objectRef)
		sharedStateData  = smachine.NewUnboundSharedData(&smObject.SharedState)
		smObjectAccessor = object.SharedStateAccessor{SharedDataLink: sharedStateData}
		smExecute        = SMExecute{
			Payload:           request,
			pulseSlot:         &pulseSlot,
			objectSharedState: smObjectAccessor,
			pcs:               virtualtestutils.GetPCS(meRef),
		}
	)

	initializedSMExecute := expectedInitState(ctx, smExecute)

	{
		initCtx := smachine.NewInitializationContextMock(mc).
			GetContextMock.Return(ctx).
			SetDefaultMigrationMock.Set(testutils.AssertMigration(t, smExecute.migrationDefault)).
			ShareMock.Return(smachine.SharedDataLink{}).
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepCheckRequest))

		smExecute.Init(initCtx)
	}

	initializedSMExecute.execution.Object = reference.Global{}
	smExecute.execution.Object = reference.Global{}

	initializedSMExecute.lmnContext = nil
	smExecute.lmnContext = nil

	assert.Equal(t, initializedSMExecute, smExecute)

	mc.Finish()
}

func TestSMExecute_StartRequestProcessing(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		ctx = instestlogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd        = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		caller    = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		callee    = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)

		callFlags = rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty)

		request = &rms.VCallRequest{
			CallType:       rms.CallTypeConstructor,
			CallFlags:      callFlags,
			Caller:         rms.NewReference(caller),
			Callee:         rms.NewReference(callee),
			CallSiteMethod: "New",
			CallOutgoing:   rms.NewReference(gen.UniqueGlobalRefWithPulse(pd.PulseNumber)),
			Arguments:      rms.NewBytes(insolar.MustSerialize([]interface{}{})),
		}

		meRef           = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		objectRef       = virtualtestutils.GetObjectReference(request, meRef)
		smObject        = object.NewStateMachineObject(objectRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)
		desc            = descriptor.NewObject(
			objectRef,
			gen.UniqueLocalRefWithPulse(pd.PulseNumber),
			reference.Global{},
			[]byte("dummy"),
			false,
		)
		smObjectAccessor = object.SharedStateAccessor{SharedDataLink: sharedStateData}
		smExecute        = SMExecute{
			Payload:           request,
			pulseSlot:         &pulseSlot,
			objectSharedState: smObjectAccessor,
		}
	)
	smObject.SetDescriptorDirty(desc)

	smExecute = expectedInitState(ctx, smExecute)
	smExecute.lmnContext = vnlmn.NewDummyRegistrationCtx(desc.State())

	smObject.SharedState.Info.KnownRequests.Add(callFlags.GetInterference(), smExecute.execution.Outgoing)

	assert.Equal(t, 0, smObject.KnownRequests.GetList(isolation.CallTolerable).CountActive())
	assert.Equal(t, 0, smObject.KnownRequests.GetList(isolation.CallIntolerable).CountActive())

	assert.Equal(t, 1, smObject.KnownRequests.Len())

	{ // updateCounters after
		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			SetDefaultMigrationMock.Return().
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepExecuteStart))

		smExecute.stepStartRequestProcessing(execCtx)
	}

	assert.Equal(t, 1, smObject.KnownRequests.GetList(isolation.CallTolerable).CountActive())
	assert.Equal(t, 0, smObject.KnownRequests.GetList(isolation.CallIntolerable).CountActive())

	assert.Equal(t, 1, smObject.KnownRequests.Len())
	assert.Equal(t, callregistry.RequestProcessing, smObject.KnownRequests.GetList(isolation.CallTolerable).GetState(smExecute.execution.Outgoing))

	mc.Finish()
}

func TestSMExecute_DeduplicationUsingPendingsTableRequestNotExist(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		ctx = instestlogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd                = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot         = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		callee            = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		caller            = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		constructorOutRef = reference.NewRecordOf(caller, gen.UniqueLocalRefWithPulse(pd.PulseNumber))
		objectRef         = reference.NewSelf(constructorOutRef.GetLocal())
		smObject          = object.NewStateMachineObject(objectRef)
		sharedStateData   = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty)
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &rms.VCallRequest{
		CallType:       rms.CallTypeConstructor,
		CallFlags:      callFlags,
		Callee:         rms.NewReference(callee),
		CallSiteMethod: "New",
		CallOutgoing:   rms.NewReference(constructorOutRef),
		Arguments:      rms.NewBytes(insolar.MustSerialize([]interface{}{})),
	}

	smExecute := SMExecute{
		Payload:           request,
		pulseSlot:         &pulseSlot,
		objectSharedState: smObjectAccessor,
	}

	smExecute = expectedInitState(ctx, smExecute)

	{
		// duplicate pending request doesnt exists
		// expect jump
		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepTakeLock))

		smExecute.stepDeduplicate(execCtx)
	}

	mc.Finish()
}

func TestSMExecute_DeduplicationUsingPendingsTableRequestExist(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		ctx = instestlogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd                = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot         = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		caller            = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		callee            = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		constructorOutRef = reference.NewRecordOf(caller, gen.UniqueLocalRefWithPulse(pd.PulseNumber.Next(1)))
		objectRef         = reference.NewSelf(constructorOutRef.GetLocal())
		smObject          = object.NewStateMachineObject(objectRef)
		sharedStateData   = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty)
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &rms.VCallRequest{
		CallType:       rms.CallTypeConstructor,
		CallFlags:      callFlags,
		CallSiteMethod: "New",
		Callee:         rms.NewReference(callee),
		CallOutgoing:   rms.NewReference(constructorOutRef),
		Arguments:      rms.NewBytes(insolar.MustSerialize([]interface{}{})),
	}

	smExecute := SMExecute{
		Payload:           request,
		pulseSlot:         &pulseSlot,
		objectSharedState: smObjectAccessor,
	}

	smExecute = expectedInitState(ctx, smExecute)

	{
		// duplicate pending request exists and is active
		// expect SM stop
		pendingList := smObject.PendingTable.GetList(isolation.CallIntolerable)
		pendingList.Add(smExecute.execution.Outgoing)

		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			StopMock.Return(smachine.StateUpdate{})

		smExecute.stepDeduplicate(execCtx)
	}

	{
		// reset tables
		smObject.KnownRequests = callregistry.NewWorkingTable()
		smObject.PendingTable = callregistry.NewRequestTable()

		// duplicate pending request exists, but is finished
		// expect jump
		pendingList := smObject.PendingTable.GetList(isolation.CallIntolerable)
		pendingList.Add(smExecute.execution.Outgoing)
		pendingList.Finish(smExecute.execution.Outgoing)

		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepDeduplicateThroughPreviousExecutor))

		smExecute.stepDeduplicate(execCtx)
	}

	mc.Finish()
}

func TestSMExecute_DeduplicateThroughPreviousExecutor(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		ctx = instestlogger.TestContext(t)
		mc  = minimock.NewController(t)

		oldPd           = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pd              = pulse.NewPulsarData(oldPd.NextPulseNumber(), oldPd.NextPulseDelta, oldPd.NextPulseDelta, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		callerRef       = gen.UniqueGlobalRefWithPulse(oldPd.PulseNumber)
		outgoingRef     = reference.NewRecordOf(callerRef, gen.UniqueLocalRefWithPulse(pd.PulseNumber))
		objectRef       = gen.UniqueGlobalRefWithPulse(oldPd.PulseNumber)
		smObject        = object.NewStateMachineObject(objectRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty)
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &rms.VCallRequest{
		CallType:       rms.CallTypeMethod,
		Callee:         rms.NewReference(objectRef),
		CallFlags:      callFlags,
		CallSiteMethod: "Method",
		CallOutgoing:   rms.NewReference(outgoingRef),
		Arguments:      rms.NewBytes(insolar.MustSerialize([]interface{}{})),
	}

	messageSender := messagesender.NewServiceMockWrapper(mc)
	messageSenderAdapter := messageSender.NewAdapterMock()
	messageSenderAdapter.SetDefaultPrepareAsyncCall(ctx)

	checkMessage := func(msg rmsreg.GoGoSerializable) {
		switch msg0 := msg.(type) {
		case *rms.VFindCallRequest:
			assert.Equal(t, oldPd.PulseNumber, msg0.LookAt)
			assert.Equal(t, objectRef, msg0.Callee.GetValue())
			assert.Equal(t, request.CallOutgoing, msg0.Outgoing)
		default:
			panic("Unexpected message type")
		}
	}
	messageSender.SendRole.SetCheckMessage(checkMessage)

	smExecute := SMExecute{
		Payload:           request,
		pulseSlot:         &pulseSlot,
		objectSharedState: smObjectAccessor,
		messageSender:     messageSenderAdapter.Mock(),
	}
	smExecute = expectedInitState(ctx, smExecute)

	{
		// expect publish bargeIn and send VFindCallRequest

		pendingList := smObject.PendingTable.GetList(isolation.CallIntolerable)
		pendingList.Add(smExecute.execution.Outgoing)

		execCtx := smachine.NewExecutionContextMock(mc).
			NewBargeInWithParamMock.Set(
			func(applyFunc smachine.BargeInApplyFunc) smachine.BargeInWithParam {
				return smachine.BargeInWithParam{}
			}).
			PublishGlobalAliasAndBargeInMock.Set(
			func(key interface{}, handler smachine.BargeInHolder) (b1 bool) {
				res, ok := key.(DeduplicationBargeInKey)
				if !ok {
					panic("Unexpected message type")
				}

				assert.Equal(t, oldPd.PulseNumber, res.LookAt)
				assert.Equal(t, smExecute.execution.Outgoing, res.Outgoing)
				assert.Equal(t, smExecute.execution.Object, res.Callee)

				return true
			}).
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepWaitFindCallResponse))

		smExecute.stepDeduplicateThroughPreviousExecutor(execCtx)
	}

	mc.Finish()
}

func TestSMExecute_ProcessFindCallResponse(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		ctx = instestlogger.TestContext(t)
		mc  = minimock.NewController(mocklog.T(t))

		oldPd           = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pd              = pulse.NewPulsarData(oldPd.NextPulseNumber(), oldPd.NextPulseDelta, oldPd.NextPulseDelta, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		callerRef       = gen.UniqueGlobalRefWithPulse(oldPd.PulseNumber)
		outgoingRef     = reference.NewRecordOf(callerRef, gen.UniqueLocalRefWithPulse(pd.PulseNumber))
		objectRef       = gen.UniqueGlobalRefWithPulse(oldPd.PulseNumber)
		smObject        = object.NewStateMachineObject(objectRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty)

		sender = gen.UniqueGlobalRef()
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &rms.VCallRequest{
		CallType:       rms.CallTypeMethod,
		Callee:         rms.NewReference(objectRef),
		CallFlags:      callFlags,
		CallSiteMethod: "Method",
		CallOutgoing:   rms.NewReference(outgoingRef),
		Arguments:      rms.NewBytes(insolar.MustSerialize([]interface{}{})),
	}

	smExecute := SMExecute{
		Payload:           request,
		pulseSlot:         &pulseSlot,
		objectSharedState: smObjectAccessor,
		Meta:              &rms.Meta{Sender: rms.NewReference(sender)},
	}

	smExecute = expectedInitState(ctx, smExecute)

	{
		smExecute.findCallResponse = &rms.VFindCallResponse{Status: rms.CallStateMissing}
		pendingList := smObject.PendingTable.GetList(isolation.CallIntolerable)
		pendingList.Add(smExecute.execution.Outgoing)

		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepTakeLock))

		smExecute.stepProcessFindCallResponse(execCtx)
	}

	{
		smExecute.findCallResponse = &rms.VFindCallResponse{Status: rms.CallStateUnknown}
		pendingList := smObject.PendingTable.GetList(isolation.CallIntolerable)
		pendingList.Add(smExecute.execution.Outgoing)

		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepTakeLock))

		smExecute.stepProcessFindCallResponse(execCtx)
	}

	{
		smExecute.findCallResponse = &rms.VFindCallResponse{
			Status:     rms.CallStateFound,
			CallResult: nil,
		}

		pendingList := smObject.PendingTable.GetList(isolation.CallIntolerable)
		pendingList.Add(smExecute.execution.Outgoing)

		execCtx := smachine.NewExecutionContextMock(mc).
			LogMock.Return(smachine.Logger{}).
			StopMock.Return(smachine.StateUpdate{})

		smExecute.stepProcessFindCallResponse(execCtx)
	}

	{
		returnArguments := []byte{1, 2, 3}
		smExecute.findCallResponse = &rms.VFindCallResponse{
			Status: rms.CallStateFound,
			CallResult: &rms.VCallResult{
				ReturnArguments: rms.NewBytes(returnArguments),
			},
		}

		messageSender := messagesender.NewServiceMockWrapper(mc)
		messageSenderAdapter := messageSender.NewAdapterMock()
		messageSenderAdapter.SetDefaultPrepareAsyncCall(ctx)
		checkMessage := func(msg rmsreg.GoGoSerializable) {
			switch msg0 := msg.(type) {
			case *rms.VCallResult:
				assert.Equal(t, returnArguments, msg0.ReturnArguments.GetBytes())
			default:
				panic("Unexpected message type")
			}
		}
		checkTarget := func(target reference.Global) {
			assert.Equal(t, smExecute.Meta.Sender.GetValue(), target)
		}

		messageSender.SendTarget.SetCheckMessage(checkMessage)
		messageSender.SendTarget.SetCheckTarget(checkTarget)

		smExecute.messageSender = messageSenderAdapter.Mock()

		execCtx := smachine.NewExecutionContextMock(mc).
			LogMock.Return(smachine.Logger{}).
			StopMock.Return(smachine.StateUpdate{})

		smExecute.stepProcessFindCallResponse(execCtx)
	}

	mc.Finish()
}

func TestSMExecute_DeduplicationForOldRequest(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		ctx = instestlogger.TestContext(t)
		mc  = minimock.NewController(t)

		oldPd           = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pd              = pulse.NewPulsarData(oldPd.NextPulseNumber(), oldPd.NextPulseDelta, oldPd.PrevPulseDelta, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		caller          = gen.UniqueGlobalRefWithPulse(oldPd.PulseNumber)
		outgoingRef     = reference.NewRecordOf(caller, gen.UniqueLocalRefWithPulse(oldPd.PulseNumber))
		objectRef       = gen.UniqueGlobalRefWithPulse(oldPd.PulseNumber)
		smObject        = object.NewStateMachineObject(objectRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty)
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &rms.VCallRequest{
		CallType:       rms.CallTypeMethod,
		Caller:         rms.NewReference(caller),
		Callee:         rms.NewReference(objectRef),
		CallFlags:      callFlags,
		CallSiteMethod: "Method",
		CallOutgoing:   rms.NewReference(outgoingRef),
		Arguments:      rms.NewBytes(insolar.MustSerialize([]interface{}{})),
	}

	smExecute := SMExecute{
		Payload:           request,
		pulseSlot:         &pulseSlot,
		objectSharedState: smObjectAccessor,

		methodIsolation: contract.MethodIsolation{
			Interference: callFlags.GetInterference(),
			State:        callFlags.GetState(),
		},
	}

	smExecute = expectedInitState(ctx, smExecute)

	{
		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			AcquireMock.Return(true).
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepDeduplicate))

		smExecute.stepIsolationNegotiation(execCtx)
	}
}

func TestSMExecute_TokenInOutgoingMessage(t *testing.T) {
	var (
		selfRef  = gen.UniqueGlobalRef()
		otherRef = gen.UniqueGlobalRef()
	)

	tests := []struct {
		name                 string
		token                rms.CallDelegationToken
		expectedTokenIsEmpty bool
	}{
		{
			name: "SelfToken",
			token: rms.CallDelegationToken{
				Caller:   rms.NewReference(selfRef),
				Approver: rms.NewReference(selfRef),
			},
			expectedTokenIsEmpty: true,
		},
		{
			name: "OtherToken",
			token: rms.CallDelegationToken{
				Caller:   rms.NewReference(selfRef),
				Approver: rms.NewReference(otherRef),
			},
			expectedTokenIsEmpty: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer executeLeakCheck(t)

			var (
				ctx = instestlogger.TestContext(t)
				mc  = minimock.NewController(t)

				pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
				pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
				caller          = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
				callee          = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
				smGlobalRef     = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
				smObject        = object.NewStateMachineObject(smGlobalRef)
				sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

				callFlags = rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty)
			)

			smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
			request := &rms.VCallRequest{
				CallType:       rms.CallTypeConstructor,
				CallFlags:      callFlags,
				Caller:         rms.NewReference(caller),
				Callee:         rms.NewReference(callee),
				CallSiteMethod: "New",
				CallOutgoing:   rms.NewReference(smGlobalRef),
				Arguments:      rms.NewBytes(insolar.MustSerialize([]interface{}{})),
			}

			affMock := affinity.NewHelperMock(t).MeMock.Return(selfRef)

			authService := authentication.NewService(ctx, affMock)

			checkMessage := func(msg rmsreg.GoGoSerializable) {
				expectedToken := rms.CallDelegationToken{}
				if !test.expectedTokenIsEmpty {
					expectedToken = test.token
				}
				switch msg0 := msg.(type) {
				case *rms.VCallResult:
					assert.Equal(t, expectedToken, msg0.DelegationSpec)
				case *rms.VDelegatedRequestFinished:
					assert.Equal(t, expectedToken, msg0.DelegationSpec)
				default:
					panic("Unexpected message type")
				}
			}

			messageSender := messagesender.NewServiceMockWrapper(mc)
			messageSender.SendRole.SetCheckMessage(checkMessage)
			messageSender.SendTarget.SetCheckMessage(checkMessage)
			messageSenderAdapter := messageSender.NewAdapterMock()
			messageSenderAdapter.SetDefaultPrepareAsyncCall(ctx)

			smExecute := SMExecute{
				Meta: &rms.Meta{
					Sender: rms.NewReference(otherRef),
				},
				Payload:               request,
				pulseSlot:             &pulseSlot,
				objectSharedState:     smObjectAccessor,
				authenticationService: authService,
				delegationTokenSpec:   test.token,
				executionNewState: &execution.Update{
					Outgoing: execution.CallMethod{},
					Result:   &requestresult.RequestResult{},
				},
				messageSender: messageSenderAdapter.Mock(),
			}

			smExecute = expectedInitState(ctx, smExecute)

			{
				execCtx := smachine.NewExecutionContextMock(mc).
					JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepFinishRequest))

				smExecute.stepSendCallResult(execCtx)
			}

			{
				execCtx := smachine.NewExecutionContextMock(mc).
					StopMock.Return(smachine.StateUpdate{})

				smExecute.stepSendDelegatedRequestFinished(execCtx)
			}

			mc.Finish()

		})
	}
}

func TestSMExecute_VCallResultPassedToSMObject(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		ctx = instestlogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		caller          = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		smGlobalRef     = gen.UniqueGlobalRefWithPulse(pd.PulseNumber)
		smObject        = object.NewStateMachineObject(smGlobalRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty)
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &rms.VCallRequest{
		CallType:       rms.CallTypeConstructor,
		CallFlags:      callFlags,
		CallSiteMethod: "New",
		CallOutgoing:   rms.NewReference(smGlobalRef),
		Callee:         rms.NewReference(gen.UniqueGlobalRefWithPulse(pd.PulseNumber)),
		Caller:         rms.NewReference(caller),
		Arguments:      rms.NewBytes(insolar.MustSerialize([]interface{}{})),
	}

	smExecute := SMExecute{
		Meta: &rms.Meta{
			Sender: rms.NewReference(gen.UniqueGlobalRef()),
		},
		Payload:           request,
		pulseSlot:         &pulseSlot,
		objectSharedState: smObjectAccessor,
		executionNewState: &execution.Update{
			Outgoing: execution.CallMethod{},
			Result:   &requestresult.RequestResult{},
		},
		messageSender: adapter.NewMessageSenderMock(t).PrepareAsyncMock.Set(func(e1 smachine.ExecutionContext, fn adapter.AsyncCallFunc) (a1 smachine.AsyncCallRequester) {
			return smachine.NewAsyncCallRequesterMock(t).WithoutAutoWakeUpMock.Set(func() (a1 smachine.AsyncCallRequester) {
				return smachine.NewAsyncCallRequesterMock(t).StartMock.Set(func() {

				})
			})
		}),
	}

	smExecute = expectedInitState(ctx, smExecute)

	smObject.KnownRequests.Add(isolation.CallTolerable, request.CallOutgoing.GetValue())
	smObject.KnownRequests.SetActive(isolation.CallTolerable, request.CallOutgoing.GetValue())

	{
		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepFinishRequest))

		smExecute.stepSendCallResult(execCtx)
	}

	{
		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			StopMock.Return(smachine.StateUpdate{})

		smExecute.stepFinishRequest(execCtx)
	}
	assert.Equal(t, 1, smObject.KnownRequests.Len())

	result, ok := smObject.KnownRequests.GetResults()[smGlobalRef]

	assert.True(t, ok, "object not in map")
	assert.NotNil(t, result)

	mc.Finish()
}

func TestSendVStateReportWithMissingState_IfConstructorWasInterruptedBeforeRunnerCall(t *testing.T) {
	defer executeLeakCheck(t)
	insrail.LogCase(t, "C5084")

	var (
		mc  = minimock.NewController(t)
		ctx = context.Background()
	)

	slotMachine := virtualdebugger.New(ctx, t)
	slotMachine.PrepareRunner(ctx, mc)

	var (
		class                                      = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		caller                                     = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		catalog     object.Catalog                 = object.NewLocalCatalog()
		authService authentication.Service         = authentication.NewServiceMock(t)
		memoryCache memoryCacheAdapter.MemoryCache = memoryCacheAdapter.NewMemoryCacheMock(t)

		limiter = tool.NewRunnerLimiter(4)
	)

	slotMachine.AddInterfaceDependency(&catalog)
	slotMachine.AddInterfaceDependency(&authService)
	slotMachine.AddInterfaceDependency(&memoryCache)
	slotMachine.AddDependency(limiter)

	outgoing := reference.NewRecordOf(caller, slotMachine.GenerateLocal())

	smExecute := SMExecute{
		Payload: &rms.VCallRequest{
			CallType:     rms.CallTypeConstructor,
			CallFlags:    rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
			CallOutgoing: rms.NewReference(outgoing),

			Caller:         rms.NewReference(caller),
			Callee:         rms.NewReference(class),
			CallSiteMethod: "New",
		},
		Meta: &rms.Meta{
			Sender: rms.NewReference(caller),
		},
	}
	meRef := gen.UniqueGlobalRef()
	objectRef := virtualtestutils.GetObjectReference(smExecute.Payload, meRef)

	slotMachine.Start()

	var vStateReportRecv = make(chan struct{})

	slotMachine.PrepareMockedMessageSender(mc)
	slotMachine.MessageSender.SendRole.SetCheckMessage(func(msg rmsreg.GoGoSerializable) {
		switch res := msg.(type) {
		case *rms.VStateReport:
			assert.Equal(t, rms.StateStatusMissing, res.Status)
			assert.Equal(t, objectRef, res.Object.GetValue())
			assert.Equal(t, int32(0), res.OrderedPendingCount)
			assert.Equal(t, int32(0), res.UnorderedPendingCount)
			assert.Empty(t, res.LatestDirtyState)
			assert.Empty(t, res.LatestValidatedState)
			close(vStateReportRecv)
		case *rms.LRegisterRequest:
			_ = res.AnticipatedRef.Get()

			key := vnlmn.ResultAwaitKey{
				AnticipatedRef: res.AnticipatedRef,
				RequiredFlag:   rms.RegistrationFlags_Fast,
			}

			_, bargeIn := slotMachine.SlotMachine.GetPublishedGlobalAliasAndBargeIn(key)
			bargeIn.CallWithParam(&rms.LRegisterResponse{
				Flags:              rms.RegistrationFlags_Fast,
				AnticipatedRef:     res.AnticipatedRef,
				RegistrarSignature: rms.NewBytes([]byte("dummy")),
			})
		default:
			t.Fatal("unreachable")
		}
	})

	smWrapper := slotMachine.AddStateMachine(ctx, &smExecute)
	slotMachine.RunTil(smWrapper.BeforeStep(smExecute.stepStartRequestProcessing))
	slotMachine.Migrate()
	go slotMachine.RunTil(func(event debuglogger.UpdateEvent) bool {
		select {
		case <-vStateReportRecv:
			return true
		default:
			return false
		}
	})
	commonTestUtils.WaitSignalsTimed(t, 10*time.Second, vStateReportRecv)

	slotMachine.Stop()
	mc.Finish()
}

func TestSMExecute_StopWithoutMessagesIfPulseChangedBeforeOutgoing(t *testing.T) {
	defer executeLeakCheck(t)
	insrail.LogCase(t, "C5101")

	const stateMemory = "213"

	var (
		mc  = minimock.NewController(t)
		ctx = context.Background()
	)

	slotMachine := virtualdebugger.New(ctx, t)
	slotMachine.InitEmptyMessageSender(mc)

	var (
		class     = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		caller    = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())
		objectRef = gen.UniqueGlobalRefWithPulse(slotMachine.PulseSlot.CurrentPulseNumber())

		catalogWrapper                                = object.NewCatalogMockWrapper(mc)
		catalog        object.Catalog                 = catalogWrapper.Mock()
		authService    authentication.Service         = authentication.NewServiceMock(t)
		memoryCache    memoryCacheAdapter.MemoryCache = memoryCacheAdapter.NewMemoryCacheMock(t)

		limiter = tool.NewRunnerLimiter(4)
	)

	slotMachine.PrepareMockedRunner(ctx, mc)
	slotMachine.AddInterfaceDependency(&catalog)
	slotMachine.AddInterfaceDependency(&authService)
	slotMachine.AddInterfaceDependency(&memoryCache)
	slotMachine.AddDependency(limiter)

	obj := object.Info{
		Reference:      objectRef,
		PendingTable:   callregistry.NewRequestTable(),
		KnownRequests:  callregistry.NewWorkingTable(),
		ReadyToWork:    smsync.NewConditional(1, "ReadyToWork").SyncLink(),
		OrderedExecute: limiter.NewChildSemaphore(1, "MutableExecution").SyncLink(),
	}
	obj.SetState(object.HasState)
	obj.SetDescriptorDirty(descriptor.NewObject(reference.Global{}, reference.Local{}, class, []byte(stateMemory), false))
	sharedStateData := smachine.NewUnboundSharedData(&object.SharedState{Info: obj})
	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}

	catalogWrapper.AddObject(objectRef, smObjectAccessor)
	catalogWrapper.AllowAccessMode(object.CatalogMockAccessGetOrCreate)

	outgoing := reference.NewRecordOf(objectRef, slotMachine.GenerateLocal())

	slotMachine.RunnerMock.AddExecutionClassify(
		outgoing,
		contract.MethodIsolation{Interference: isolation.CallTolerable, State: isolation.CallDirty},
		nil,
	)

	smExecute := SMExecute{
		Payload: &rms.VCallRequest{
			CallType:       rms.CallTypeMethod,
			Caller:         rms.NewReference(caller),
			Callee:         rms.NewReference(objectRef),
			CallFlags:      rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
			CallOutgoing:   rms.NewReference(outgoing),
			CallSiteMethod: "test",
			Arguments:      rms.NewBytes(insolar.MustSerialize([]interface{}{})),
		},
		Meta: &rms.Meta{
			Sender: rms.NewReference(caller),
		},
	}
	slotMachine.Start()
	defer slotMachine.Stop()

	smWrapper := slotMachine.AddStateMachine(ctx, &smExecute)
	slotMachine.RunTil(smWrapper.BeforeStep(smExecute.stepStartRequestProcessing))
	slotMachine.Migrate()
	slotMachine.RunTil(smWrapper.AfterStop())

	report := obj.BuildStateReport()
	assert.Equal(t, objectRef, report.Object.GetValue())
	assert.Equal(t, rms.StateStatusReady, report.Status)
	assert.Equal(t, int32(0), report.OrderedPendingCount)
	assert.Equal(t, int32(0), report.UnorderedPendingCount)
	state := obj.BuildLatestDirtyState()
	assert.Equal(t, []byte(stateMemory), state.Memory.GetBytes())
	assert.False(t, state.Deactivated)

	mc.Finish()
}
