// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executionevent"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils/stepchecker"
)

func TestSMExecute_MigrationDuringSendOutgoing(t *testing.T) {
	var (
		ctx = inslogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd        = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		// catalog         = object.NewCatalogMockWrapper(mc)
		smObjectID = gen.UniqueIDWithPulse(pd.PulseNumber)
		// smGlobalRef     = reference.NewSelf(smObjectID)
		// smObject        = object.NewStateMachineObject(smGlobalRef)
		// sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

		callFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)
	)
	defer mc.Finish()

	// smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}
	// catalog.AddObject(smGlobalRef, smObjectAccessor)

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			Polymorph:           uint32(payload.TypeVCallRequest),
			CallType:            payload.CTConstructor,
			CallFlags:           callFlags,
			CallSiteDeclaration: testwallet.GetPrototype(),
			CallSiteMethod:      "New",
			CallOutgoing:        smObjectID,
			Arguments:           insolar.MustSerialize([]interface{}{}),
		},
		// objectCatalog: catalog.Mock(),
		pulseSlot: &pulseSlot,
		executionNewState: &executionupdate.ContractExecutionStateUpdate{
			Outgoing: executionevent.CallMethod{},
		},
		messageSender: adapter.NewMessageSenderMock(t).PrepareAsyncMock.Set(func(e1 smachine.ExecutionContext, fn adapter.AsyncCallFunc) (a1 smachine.AsyncCallRequester) {
			return smachine.NewAsyncCallRequesterMock(t).WithoutAutoWakeUpMock.Set(func() (a1 smachine.AsyncCallRequester) {
				return smachine.NewAsyncCallRequesterMock(t).StartMock.Set(func() {

				})
			})
		}),
	}

	stepChecker := stepchecker.New()
	{
		exec := SMExecute{}
		stepChecker.AddStep(exec.stepCheckRequest)
		stepChecker.AddStep(exec.stepSendOutgoing)
		stepChecker.AddStep(exec.stepSendOutgoing)
	}
	defer func() { require.NoError(t, stepChecker.CheckDone()) }()

	{
		execCtx := smachine.NewExecutionContextMock(mc).
			GetContextMock.Return(ctx).JumpMock.Set(stepChecker.CheckJumpW(t))
		smExecute.Init(execCtx)
	}

	{
		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smExecute.stepExecuteOutgoing(execCtx)
	}

	{
		execCtx := smachine.NewExecutionContextMock(mc).
			NewBargeInWithParamMock.Set(
			func(applyFunc smachine.BargeInApplyFunc) smachine.BargeInWithParam {
				return smachine.BargeInWithParam{}
			}).
			PublishGlobalAliasAndBargeInMock.Set(
			func(key interface{}, handler smachine.BargeInHolder) (b1 bool) {
				return true
			}).SleepMock.Set(
			func() (c1 smachine.ConditionalBuilder) {
				return smachine.NewStateConditionalBuilderMock(t).
					ThenJumpExtMock.Set(
					func(s1 smachine.SlotStep) (s2 smachine.StateUpdate) {
						require.Equal(t, runtime.FuncForPC(reflect.ValueOf(smExecute.stepExecuteContinue).Pointer()).Name(), runtime.FuncForPC(reflect.ValueOf(s1.Transition).Pointer()).Name())
						require.Equal(t, runtime.FuncForPC(reflect.ValueOf(smExecute.migrateDuringSendOutgoing).Pointer()).Name(), runtime.FuncForPC(reflect.ValueOf(s1.Migration).Pointer()).Name())
						return smachine.StateUpdate{}
					})
			})

		smExecute.stepSendOutgoing(execCtx)
	}

	{ // check migration is successful
		migrationCtx := smachine.NewMigrationContextMock(mc).
			// AffectedStepMock.Return(smachine.SlotStep{Transition: smExecute.stepSendOutgoing}).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smExecute.migrateDuringSendOutgoing(migrationCtx)

		require.Equal(t, true, smExecute.migrationHappened)
		require.Equal(t,
			payload.BuildCallRequestFlags(contract.SendResultDefault, contract.RepeatedCall),
			smExecute.outgoing.CallRequestFlags,
		)
	}

	{
		execCtx := smachine.NewExecutionContextMock(mc).
			SleepMock.Set(
			func() (c1 smachine.ConditionalBuilder) {
				return smachine.NewStateConditionalBuilderMock(t).
					ThenJumpExtMock.Set(
					func(s1 smachine.SlotStep) (s2 smachine.StateUpdate) {
						require.Equal(t, runtime.FuncForPC(reflect.ValueOf(smExecute.stepExecuteContinue).Pointer()).Name(), runtime.FuncForPC(reflect.ValueOf(s1.Transition).Pointer()).Name())
						require.Equal(t, runtime.FuncForPC(reflect.ValueOf(smExecute.migrateDuringSendOutgoing).Pointer()).Name(), runtime.FuncForPC(reflect.ValueOf(s1.Migration).Pointer()).Name())
						return smachine.StateUpdate{}
					})
			})

		smExecute.stepSendOutgoing(execCtx)
	}

}
