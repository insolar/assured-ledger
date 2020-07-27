// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	commonTestUtils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/testutils/mocklog"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callregistry"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	virtualTestUtils "github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/shareddata"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/slotdebugger"
	"github.com/insolar/assured-ledger/ledger-core/virtual/tool"
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

	if sm.Payload.CallType == payload.CTConstructor {
		sm.isConstructor = true
		sm.execution.Object = reference.NewSelf(sm.Payload.CallOutgoing.GetLocal())
	} else {
		sm.execution.Object = sm.Payload.Callee
	}

	sm.execution.Incoming = reference.NewRecordOf(sm.Payload.Callee, sm.Payload.CallOutgoing.GetLocal())
	sm.execution.Outgoing = sm.Payload.CallOutgoing

	sm.execution.Isolation = contract.MethodIsolation{
		Interference: sm.Payload.CallFlags.GetInterference(),
		State:        sm.Payload.CallFlags.GetState(),
	}
	return sm
}

func TestSMExecute_Init(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		ctx = instestlogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID      = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
		caller          = gen.UniqueGlobalRef()
		smGlobalRef     = reference.NewRecordOf(caller, smObjectID)
		smObject        = object.NewStateMachineObject(smGlobalRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &payload.VCallRequest{
		CallType:            payload.CTConstructor,
		CallFlags:           callFlags,
		CallSiteDeclaration: testwallet.GetClass(),
		CallSiteMethod:      "New",
		Caller:              caller,
		CallOutgoing:        smGlobalRef,
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	smExecute := SMExecute{
		Payload:           request,
		pulseSlot:         &pulseSlot,
		objectSharedState: smObjectAccessor,
	}

	initializedSMExecute := expectedInitState(ctx, smExecute)

	{
		initCtx := smachine.NewInitializationContextMock(mc).
			GetContextMock.Return(ctx).
			SetDefaultMigrationMock.Set(testutils.AssertMigration(t, smExecute.migrationDefault)).
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepCheckRequest))

		smExecute.Init(initCtx)
	}

	require.Equal(t, initializedSMExecute, smExecute)

	mc.Finish()
}

func TestSMExecute_StartRequestProcessing(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		ctx = instestlogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID      = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
		caller          = gen.UniqueGlobalRef()
		smGlobalRef     = reference.NewRecordOf(caller, smObjectID)
		smObject        = object.NewStateMachineObject(smGlobalRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &payload.VCallRequest{
		CallType:            payload.CTConstructor,
		CallFlags:           callFlags,
		Caller:              caller,
		CallSiteDeclaration: testwallet.GetClass(),
		CallSiteMethod:      "New",
		CallOutgoing:        smGlobalRef,
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	smExecute := SMExecute{
		Payload:           request,
		pulseSlot:         &pulseSlot,
		objectSharedState: smObjectAccessor,
	}

	smExecute = expectedInitState(ctx, smExecute)

	smObject.SharedState.Info.KnownRequests.Add(callFlags.GetInterference(), smExecute.execution.Outgoing)

	assert.Equal(t, uint8(0), smObject.PotentialOrderedPendingCount)
	assert.Equal(t, uint8(0), smObject.PotentialUnorderedPendingCount)

	assert.Equal(t, 1, smObject.KnownRequests.Len())

	{ // updateCounters after
		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			SetDefaultMigrationMock.Return().
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepExecuteStart))

		smExecute.stepStartRequestProcessing(execCtx)
	}

	assert.Equal(t, uint8(1), smObject.PotentialOrderedPendingCount)
	assert.Equal(t, uint8(0), smObject.PotentialUnorderedPendingCount)

	assert.Equal(t, 1, smObject.KnownRequests.Len())
	assert.Equal(t, callregistry.RequestProcessing, smObject.KnownRequests.GetList(contract.CallTolerable).GetState(smExecute.execution.Outgoing))

	mc.Finish()
}

func TestSMExecute_DeduplicationUsingPendingsTableRequestNotExist(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		ctx = instestlogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd                = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot         = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		caller            = gen.UniqueGlobalRef()
		constructorOutRef = reference.NewRecordOf(caller, gen.UniqueLocalRefWithPulse(pd.PulseNumber))
		objectRef         = reference.NewSelf(constructorOutRef.GetLocal())
		smObject          = object.NewStateMachineObject(objectRef)
		sharedStateData   = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty)
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &payload.VCallRequest{
		CallType:            payload.CTConstructor,
		CallFlags:           callFlags,
		CallSiteDeclaration: testwallet.GetClass(),
		CallSiteMethod:      "New",
		CallOutgoing:        constructorOutRef,
		Arguments:           insolar.MustSerialize([]interface{}{}),
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
		caller            = gen.UniqueGlobalRef()
		constructorOutRef = reference.NewRecordOf(caller, gen.UniqueLocalRefWithPulse(pd.PulseNumber.Next(1)))
		objectRef         = reference.NewSelf(constructorOutRef.GetLocal())
		smObject          = object.NewStateMachineObject(objectRef)
		sharedStateData   = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty)
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &payload.VCallRequest{
		CallType:            payload.CTConstructor,
		CallFlags:           callFlags,
		CallSiteDeclaration: testwallet.GetClass(),
		CallSiteMethod:      "New",
		CallOutgoing:        constructorOutRef,
		Arguments:           insolar.MustSerialize([]interface{}{}),
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
		pendingList := smObject.PendingTable.GetList(contract.CallIntolerable)
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
		pendingList := smObject.PendingTable.GetList(contract.CallIntolerable)
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
		callerRef       = gen.UniqueGlobalRef()
		outgoingRef     = reference.NewRecordOf(callerRef, gen.UniqueLocalRefWithPulse(pd.PulseNumber))
		objectRef       = gen.UniqueGlobalRef()
		smObject        = object.NewStateMachineObject(objectRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty)
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &payload.VCallRequest{
		CallType:            payload.CTMethod,
		Callee:              objectRef,
		CallFlags:           callFlags,
		CallSiteDeclaration: testwallet.GetClass(),
		CallSiteMethod:      "Method",
		CallOutgoing:        outgoingRef,
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	messageSender := messagesender.NewServiceMockWrapper(mc)
	messageSenderAdapter := messageSender.NewAdapterMock()
	messageSenderAdapter.SetDefaultPrepareAsyncCall(ctx)

	checkMessage := func(msg payload.Marshaler) {
		switch msg0 := msg.(type) {
		case *payload.VFindCallRequest:
			require.Equal(t, oldPd.PulseNumber, msg0.LookAt)
			require.Equal(t, objectRef, msg0.Callee)
			require.Equal(t, request.CallOutgoing, msg0.Outgoing)
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

		pendingList := smObject.PendingTable.GetList(contract.CallIntolerable)
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

				require.Equal(t, oldPd.PulseNumber, res.LookAt)
				require.Equal(t, smExecute.execution.Outgoing, res.Outgoing)
				require.Equal(t, smExecute.execution.Object, res.Callee)

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
		callerRef       = gen.UniqueGlobalRef()
		outgoingRef     = reference.NewRecordOf(callerRef, gen.UniqueLocalRefWithPulse(pd.PulseNumber))
		objectRef       = gen.UniqueGlobalRef()
		smObject        = object.NewStateMachineObject(objectRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty)

		sender = gen.UniqueGlobalRef()
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &payload.VCallRequest{
		CallType:            payload.CTMethod,
		Callee:              objectRef,
		CallFlags:           callFlags,
		CallSiteDeclaration: testwallet.GetClass(),
		CallSiteMethod:      "Method",
		CallOutgoing:        outgoingRef,
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	smExecute := SMExecute{
		Payload:           request,
		pulseSlot:         &pulseSlot,
		objectSharedState: smObjectAccessor,
		Meta:              &payload.Meta{Sender: sender},
	}

	smExecute = expectedInitState(ctx, smExecute)

	{
		smExecute.findCallResponse = &payload.VFindCallResponse{Status: payload.MissingCall}
		pendingList := smObject.PendingTable.GetList(contract.CallIntolerable)
		pendingList.Add(smExecute.execution.Outgoing)

		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepTakeLock))

		smExecute.stepProcessFindCallResponse(execCtx)
	}

	{
		smExecute.findCallResponse = &payload.VFindCallResponse{Status: payload.UnknownCall}
		pendingList := smObject.PendingTable.GetList(contract.CallIntolerable)
		pendingList.Add(smExecute.execution.Outgoing)

		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepTakeLock))

		smExecute.stepProcessFindCallResponse(execCtx)
	}

	{
		smExecute.findCallResponse = &payload.VFindCallResponse{
			Status:     payload.FoundCall,
			CallResult: nil,
		}

		pendingList := smObject.PendingTable.GetList(contract.CallIntolerable)
		pendingList.Add(smExecute.execution.Outgoing)

		execCtx := smachine.NewExecutionContextMock(mc).
			LogMock.Return(smachine.Logger{}).
			StopMock.Return(smachine.StateUpdate{})

		smExecute.stepProcessFindCallResponse(execCtx)
	}

	{
		returnArguments := []byte{1, 2, 3}
		smExecute.findCallResponse = &payload.VFindCallResponse{
			Status: payload.FoundCall,
			CallResult: &payload.VCallResult{
				ReturnArguments: returnArguments,
			},
		}

		messageSender := messagesender.NewServiceMockWrapper(mc)
		messageSenderAdapter := messageSender.NewAdapterMock()
		messageSenderAdapter.SetDefaultPrepareAsyncCall(ctx)
		checkMessage := func(msg payload.Marshaler) {
			switch msg0 := msg.(type) {
			case *payload.VCallResult:
				require.Equal(t, returnArguments, msg0.ReturnArguments)
			default:
				panic("Unexpected message type")
			}
		}
		checkTarget := func(target reference.Global) {
			require.Equal(t, smExecute.Meta.Sender, target)
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
		caller          = gen.UniqueGlobalRef()
		outgoingRef     = reference.NewRecordOf(caller, gen.UniqueLocalRefWithPulse(oldPd.PulseNumber))
		objectRef       = gen.UniqueGlobalRef()
		smObject        = object.NewStateMachineObject(objectRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty)
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &payload.VCallRequest{
		CallType:            payload.CTMethod,
		Caller:              caller,
		Callee:              objectRef,
		CallFlags:           callFlags,
		CallSiteDeclaration: testwallet.GetClass(),
		CallSiteMethod:      "Method",
		CallOutgoing:        outgoingRef,
		Arguments:           insolar.MustSerialize([]interface{}{}),
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
		token                payload.CallDelegationToken
		expectedTokenIsEmpty bool
	}{
		{
			name: "SelfToken",
			token: payload.CallDelegationToken{
				Caller:   selfRef,
				Approver: selfRef,
			},
			expectedTokenIsEmpty: true,
		},
		{
			name: "OtherToken",
			token: payload.CallDelegationToken{
				Caller:   selfRef,
				Approver: otherRef,
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
				smObjectID      = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
				caller          = gen.UniqueGlobalRef()
				smGlobalRef     = reference.NewRecordOf(caller, smObjectID)
				smObject        = object.NewStateMachineObject(smGlobalRef)
				sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

				callFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)
			)

			smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
			request := &payload.VCallRequest{
				CallType:            payload.CTConstructor,
				CallFlags:           callFlags,
				Caller:              caller,
				CallSiteDeclaration: testwallet.GetClass(),
				CallSiteMethod:      "New",
				CallOutgoing:        smGlobalRef,
				Arguments:           insolar.MustSerialize([]interface{}{}),
			}

			affMock := affinity.NewHelperMock(t).MeMock.Return(selfRef)

			authService := authentication.NewService(ctx, affMock)

			checkMessage := func(msg payload.Marshaler) {
				expectedToken := payload.CallDelegationToken{}
				if !test.expectedTokenIsEmpty {
					expectedToken = test.token
				}
				switch msg0 := msg.(type) {
				case *payload.VCallResult:
					assert.Equal(t, expectedToken, msg0.DelegationSpec)
				case *payload.VDelegatedRequestFinished:
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
				Meta: &payload.Meta{
					Sender: otherRef,
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
		smObjectID      = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
		caller          = gen.UniqueGlobalRef()
		smGlobalRef     = reference.NewRecordOf(caller, smObjectID)
		smObject        = object.NewStateMachineObject(smGlobalRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &payload.VCallRequest{
		CallType:            payload.CTConstructor,
		CallFlags:           callFlags,
		CallSiteDeclaration: testwallet.GetClass(),
		CallSiteMethod:      "New",
		CallOutgoing:        smGlobalRef,
		Callee:              gen.UniqueGlobalRef(),
		Caller:              caller,
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	smExecute := SMExecute{
		Meta: &payload.Meta{
			Sender: gen.UniqueGlobalRef(),
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

	smObject.KnownRequests.Add(contract.CallTolerable, request.CallOutgoing)
	smObject.KnownRequests.SetActive(contract.CallTolerable, request.CallOutgoing)

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
	require.Equal(t, 1, smObject.KnownRequests.Len())

	result, ok := smObject.KnownRequests.GetResults()[smGlobalRef]

	require.True(t, ok, "object not in map")
	require.NotNil(t, result)

	mc.Finish()
}

func TestSendVStateReportWithMissingState_IfConstructorWasInterruptedBeforeRunnerCall(t *testing.T) {
	defer executeLeakCheck(t)

	t.Log("C5084")

	var (
		mc  = minimock.NewController(t)
		ctx = context.Background()

		class                              = gen.UniqueGlobalRef()
		caller                             = gen.UniqueGlobalRef()
		catalog     object.Catalog         = object.NewLocalCatalog()
		authService authentication.Service = authentication.NewServiceMock(t)

		limiter = tool.NewRunnerLimiter(4)
	)

	slotMachine := slotdebugger.New(ctx, t)
	slotMachine.PrepareRunner(ctx, mc)

	slotMachine.AddInterfaceDependency(&catalog)
	slotMachine.AddInterfaceDependency(&authService)
	slotMachine.AddDependency(limiter)

	outgoing := reference.NewRecordOf(caller, slotMachine.GenerateLocal())

	var vStateReportRecv = make(chan struct{})
	slotMachine.PrepareMockedMessageSender(mc)
	slotMachine.MessageSender.SendRole.SetCheckMessage(func(msg payload.Marshaler) {
		res, ok := msg.(*payload.VStateReport)
		require.True(t, ok)
		assert.Equal(t, payload.Missing, res.Status)
		assert.Equal(t, reference.NewSelf(outgoing.GetLocal()), res.Object)
		assert.Equal(t, int32(0), res.OrderedPendingCount)
		assert.Equal(t, int32(0), res.UnorderedPendingCount)
		assert.Empty(t, res.LatestDirtyState)
		assert.Empty(t, res.LatestValidatedState)
		close(vStateReportRecv)
	})

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			CallType:     payload.CTConstructor,
			CallFlags:    payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			CallOutgoing: outgoing,

			Caller:         caller,
			Callee:         class,
			CallSiteMethod: "New",
		},
		Meta: &payload.Meta{
			Sender: caller,
		},
	}
	slotMachine.Start()
	defer slotMachine.Stop()

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
	virtualTestUtils.WaitSignalsTimed(t, 10*time.Second, vStateReportRecv)

	mc.Finish()
}

func TestSMExecute_StopWithoutMessagesIfPulseChangedBeforeOutgoing(t *testing.T) {
	defer executeLeakCheck(t)

	t.Log("C5101")
	const stateMemory = "213"

	var (
		mc  = minimock.NewController(t)
		ctx = context.Background()

		class     = gen.UniqueGlobalRef()
		caller    = gen.UniqueGlobalRef()
		objectRef = gen.UniqueGlobalRef()

		catalog     object.Catalog         = object.NewLocalCatalog()
		authService authentication.Service = authentication.NewServiceMock(t)

		limiter = tool.NewRunnerLimiter(4)
	)

	slotMachine := slotdebugger.New(ctx, t)
	slotMachine.PrepareMockedMessageSender(mc)
	slotMachine.PrepareMockedRunner(ctx, mc)
	slotMachine.AddInterfaceDependency(&catalog)
	slotMachine.AddInterfaceDependency(&authService)
	slotMachine.AddDependency(limiter)

	var vStateReportRecv = make(chan struct{})
	checkMessage := func(msg payload.Marshaler) {
		res, ok := msg.(*payload.VStateReport)
		require.True(t, ok)
		assert.Equal(t, objectRef, res.Object)
		assert.Equal(t, payload.Ready, res.Status)
		assert.Equal(t, int32(0), res.OrderedPendingCount)
		assert.Equal(t, int32(0), res.UnorderedPendingCount)
		assert.Equal(t, []byte(stateMemory), res.ProvidedContent.LatestDirtyState.State)
		close(vStateReportRecv)
	}
	slotMachine.MessageSender.SendRole.SetCheckMessage(checkMessage)

	outgoing := reference.NewRecordOf(objectRef, slotMachine.GenerateLocal())

	slotMachine.RunnerMock.AddExecutionClassify(
		outgoing.String(),
		contract.MethodIsolation{Interference: contract.CallTolerable, State: contract.CallDirty},
		nil,
	)

	smObject := object.NewStateMachineObject(objectRef)
	smObject.SetState(object.HasState)
	smObject.SetDescriptorDirty(descriptor.NewObject(reference.Global{}, reference.Local{}, class, []byte(stateMemory)))
	slotMachine.AddStateMachine(ctx, smObject)

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			CallType:            payload.CTMethod,
			Caller:              caller,
			Callee:              objectRef,
			CallFlags:           payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			CallOutgoing:        outgoing,
			CallSiteDeclaration: class,
			CallSiteMethod:      "test",
			Arguments:           insolar.MustSerialize([]interface{}{}),
		},
		Meta: &payload.Meta{
			Sender: caller,
		},
	}
	slotMachine.Start()
	defer slotMachine.Stop()

	smWrapper := slotMachine.AddStateMachine(ctx, &smExecute)
	slotMachine.RunTil(smWrapper.BeforeStep(smExecute.stepStartRequestProcessing))
	slotMachine.Migrate()
	slotMachine.RunTil(smWrapper.AfterStop())

	go slotMachine.RunTil(func(event debuglogger.UpdateEvent) bool {
		select {
		case <-vStateReportRecv:
			return true
		default:
			return false
		}
	})
	virtualTestUtils.WaitSignalsTimed(t, 10*time.Second, vStateReportRecv)

	mc.Finish()
}
