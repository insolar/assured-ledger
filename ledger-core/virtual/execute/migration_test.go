// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/stepchecker"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
)

func TestSMExecute_MigrationDuringSendOutgoing(t *testing.T) {
	var (
		ctx = instestlogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd         = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot  = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		smObjectID = gen.UniqueLocalRefWithPulse(pd.PulseNumber)

		callFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)
	)
	defer mc.Finish()

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		MeMock.Return(gen.UniqueGlobalRef())

	smExecute := SMExecute{
		Payload: &payload.VCallRequest{
			CallType:            payload.CTConstructor,
			CallFlags:           callFlags,
			CallSiteDeclaration: testwallet.GetClass(),
			CallSiteMethod:      "New",
			CallOutgoing:        smObjectID,
			Arguments:           insolar.MustSerialize([]interface{}{}),
		},
		pulseSlot: &pulseSlot,
		executionNewState: &execution.Update{
			Outgoing: execution.CallMethod{},
		},
		authenticationService: authentication.NewService(ctx, jetCoordinatorMock),
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
		stepChecker.AddStep(exec.stepGetDelegationToken)
	}
	defer func() { require.NoError(t, stepChecker.CheckDone()) }()

	{
		initCtx := smachine.NewInitializationContextMock(mc).
			GetContextMock.Return(ctx).JumpMock.Set(stepChecker.CheckJumpW(t)).
			SetDefaultMigrationMock.Return()
		smExecute.Init(initCtx)
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
					ThenJumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepExecuteContinue))
			})

		smExecute.stepSendOutgoing(execCtx)
	}

	{ // check migration is successful
		migrationCtx := smachine.NewMigrationContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t)).AffectedStepMock.Return(smachine.SlotStep{})

		smExecute.migrateDuringExecution(migrationCtx)

		require.Equal(t, true, smExecute.migrationHappened)
	}

	{ // check step after migration
		execCtx := smachine.NewExecutionContextMock(mc).
			SleepMock.Set(
			func() (c1 smachine.ConditionalBuilder) {
				return smachine.NewStateConditionalBuilderMock(t).
					ThenJumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepExecuteContinue))
			})

		smExecute.stepSendOutgoing(execCtx)
	}
}
