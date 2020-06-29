// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/shareddata"
)

func expectedInitState(ctx context.Context, sm SMExecute) SMExecute {
	sm.execution.Context = ctx
	sm.execution.Sequence = 0
	sm.execution.Request = sm.Payload
	sm.execution.Pulse = sm.pulseSlot.PulseData()

	if sm.Payload.CallType == payload.CTConstructor {
		sm.isConstructor = true
		sm.execution.Object = reference.NewSelf(sm.Payload.CallOutgoing)
	} else {
		sm.execution.Object = sm.Payload.Callee
	}

	sm.execution.Incoming = reference.NewRecordOf(sm.Payload.Caller, sm.Payload.CallOutgoing)
	sm.execution.Outgoing = reference.NewRecordOf(sm.Payload.Callee, sm.Payload.CallOutgoing)

	sm.execution.Isolation = contract.MethodIsolation{
		Interference: sm.Payload.CallFlags.GetInterference(),
		State:        sm.Payload.CallFlags.GetState(),
	}
	return sm
}

func TestSMExecute_Init(t *testing.T) {
	var (
		ctx = inslogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID      = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
		smGlobalRef     = reference.NewSelf(smObjectID)
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
		CallOutgoing:        smObjectID,
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
	var (
		ctx = inslogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID      = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
		smGlobalRef     = reference.NewSelf(smObjectID)
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
		CallOutgoing:        smObjectID,
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	smExecute := SMExecute{
		Payload:           request,
		pulseSlot:         &pulseSlot,
		objectSharedState: smObjectAccessor,
	}

	smExecute = expectedInitState(ctx, smExecute)

	smObject.SharedState.Info.KnownRequests.GetList(callFlags.GetInterference()).Add(smExecute.execution.Outgoing)

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
	assert.Equal(t, object.RequestProcessing, smObject.KnownRequests.GetList(contract.CallTolerable).GetState(smExecute.execution.Outgoing))

	mc.Finish()
}

func TestSMExecute_DeduplicationUsingPendingsTable(t *testing.T) {
	var (
		ctx = inslogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID      = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
		smGlobalRef     = reference.NewSelf(smObjectID)
		smObject        = object.NewStateMachineObject(smGlobalRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty)
	)

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &payload.VCallRequest{
		CallType:            payload.CTConstructor,
		CallFlags:           callFlags,
		CallSiteDeclaration: testwallet.GetClass(),
		CallSiteMethod:      "New",
		CallOutgoing:        smObjectID,
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	smExecute := SMExecute{
		Payload:           request,
		pulseSlot:         &pulseSlot,
		objectSharedState: smObjectAccessor,
	}

	smExecute = expectedInitState(ctx, smExecute)

	{
		// duplicate pending request exists
		// expect SM stop
		pendingList := smObject.PendingTable.GetList(contract.CallIntolerable)
		pendingList.Add(smExecute.execution.Outgoing)

		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			LogMock.Return(smachine.Logger{}).
			StopMock.Return(smachine.StateUpdate{})

		smExecute.stepDeduplicateUsingPendingsTable(execCtx)
	}

	{
		// start deduplication before getting all pending requests
		// expecting going sleep
		smObject.PendingTable = object.NewRequestTable()

		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			AcquireForThisStepMock.Return(false).
			SleepMock.Return(
			smachine.NewStateConditionalBuilderMock(mc).
				ThenRepeatMock.Return(smachine.StateUpdate{}),
		)

		smExecute.stepDeduplicateUsingPendingsTable(execCtx)
	}

	{
		// start deduplication after getting all pending requests
		// expecting jump
		execCtx := smachine.NewExecutionContextMock(mc).
			UseSharedMock.Set(shareddata.CallSharedDataAccessor).
			AcquireForThisStepMock.Return(true).
			JumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepTakeLock))

		smExecute.stepDeduplicateUsingPendingsTable(execCtx)
	}

	mc.Finish()
}

func TestSMExecute_DeduplicationForOldRequest(t *testing.T) {
	var (
		ctx = inslogger.TestContext(t)
		mc  = minimock.NewController(t)

		oldPd           = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pd              = pulse.NewPulsarData(oldPd.NextPulseNumber(), oldPd.NextPulseDelta, oldPd.PrevPulseDelta, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		outgoingRef     = gen.UniqueLocalRefWithPulse(oldPd.PulseNumber)
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
			var (
				ctx = inslogger.TestContext(t)
				mc  = minimock.NewController(t)

				pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
				pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
				smObjectID      = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
				smGlobalRef     = reference.NewSelf(smObjectID)
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
				CallOutgoing:        smObjectID,
				Arguments:           insolar.MustSerialize([]interface{}{}),
			}

			affMock := jet.NewAffinityHelperMock(t).MeMock.Return(selfRef)

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
	var (
		ctx = inslogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID      = gen.UniqueLocalRefWithPulse(pd.PulseNumber)
		smGlobalRef     = reference.NewSelf(smObjectID)
		smObject        = object.NewStateMachineObject(smGlobalRef)
		sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)
	)

	ref := gen.UniqueGlobalRef()

	smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	request := &payload.VCallRequest{
		CallType:            payload.CTConstructor,
		CallFlags:           callFlags,
		CallSiteDeclaration: testwallet.GetClass(),
		CallSiteMethod:      "New",
		CallOutgoing:        smObjectID,
		Callee:              ref,
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

	ref = reference.NewRecordOf(request.Callee, request.CallOutgoing)

	smExecute = expectedInitState(ctx, smExecute)

	smObject.KnownRequests.GetList(contract.CallTolerable).Add(ref)

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

	res := smObject.KnownRequests.GetList(contract.CallTolerable)

	require.Equal(t, 1, res.Count())

	result, ok := res.GetResult(ref)

	require.True(t, ok)
	require.NotNil(t, result)

	mc.Finish()
}
