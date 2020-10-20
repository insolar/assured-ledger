// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package execute

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/testutils/stepchecker"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	virtualtestutils "github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

func TestSMExecute_MigrationDuringSendOutgoing(t *testing.T) {
	defer executeLeakCheck(t)

	var (
		ctx = instestlogger.TestContext(t)
		mc  = minimock.NewController(t)

		pd        = pulse.NewFirstPulsarData(10, longbits.Bits256{})
		pulseSlot = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
		meRef     = gen.UniqueGlobalRef()

		callFlags = rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty)
	)

	jetCoordinatorMock := affinity.NewHelperMock(t).MeMock.Return(meRef)

	pl := &rms.VCallRequest{
		CallType:       rms.CallTypeConstructor,
		Callee:         rms.NewReference(gen.UniqueGlobalRefWithPulse(pd.PulseNumber)),
		Caller:         rms.NewReference(gen.UniqueGlobalRefWithPulse(pd.PulseNumber)),
		CallFlags:      callFlags,
		CallSiteMethod: "New",
		CallOutgoing:   rms.NewReference(gen.UniqueGlobalRefWithPulse(pd.PulseNumber)),
		Arguments:      rms.NewBytes(insolar.MustSerialize([]interface{}{})),
	}

	// objectID := getObjectReference(pl, meRef)

	builder := execution.NewRPCBuilder(pl.CallOutgoing.GetValue(), pl.Callee.GetValue())
	callMethod := builder.CallMethod(
		gen.UniqueGlobalRefWithPulse(pd.PulseNumber),
		gen.UniqueGlobalRefWithPulse(pd.PulseNumber),
		"Method", pl.Arguments.GetBytes(),
	)

	sender := messagesender.NewServiceMockWrapper(mc)
	sender.SendRole.SetCheckPulseNumber(func(number pulse.Number) {})
	senderAdapter := sender.NewAdapterMock()
	senderAdapter.SetDefaultPrepareAsyncCall(ctx)

	smExecute := SMExecute{
		Payload:   pl,
		pulseSlot: &pulseSlot,
		executionNewState: &execution.Update{
			Type:     execution.OutgoingCall,
			Outgoing: callMethod,
		},
		authenticationService: authentication.NewService(ctx, jetCoordinatorMock),
		messageSender:         senderAdapter.Mock(),
		referenceBuilder:      virtualtestutils.GetReferenceBuilder(meRef),
	}

	stepChecker := stepchecker.New()
	{
		exec := SMExecute{}
		stepChecker.AddStep(exec.stepCheckRequest)
		stepChecker.AddStep(exec.stepRegisterOutgoing)
		stepChecker.AddStep(exec.stepGetDelegationToken)
	}
	defer func() { assert.NoError(t, stepChecker.CheckDone()) }()

	{
		initCtx := smachine.NewInitializationContextMock(mc).
			GetContextMock.Return(ctx).JumpMock.Set(stepChecker.CheckJumpW(t)).
			SetDefaultMigrationMock.Return().ShareMock.Return(smachine.SharedDataLink{})

		smExecute.Init(initCtx)
	}

	{
		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t))

		smExecute.stepExecuteOutgoing(execCtx)
	}

	{
		execCtx := smachine.NewExecutionContextMock(mc).
			NewBargeInWithParamMock.Return(smachine.BargeInWithParam{}).
			PublishGlobalAliasAndBargeInMock.Return(true).
			SleepMock.Set(
			func() (c1 smachine.ConditionalBuilder) {
				return smachine.NewStateConditionalBuilderMock(t).
					ThenJumpMock.Set(testutils.AssertJumpStep(t, smExecute.stepWaitAndRegisterOutgoingResult))
			})

		smExecute.stepSendOutgoing(execCtx)
	}

	{ // check migration is successful
		migrationCtx := smachine.NewMigrationContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t)).
			AffectedStepMock.Return(smachine.SlotStep{})

		smExecute.migrateDuringExecution(migrationCtx)

		require.Equal(t, true, smExecute.migrationHappened)
	}

	{ // check step after migration
		execCtx := smachine.NewExecutionContextMock(mc).
			SleepMock.Set(testutils.AssertConditionalBuilderJumpStep(t, smExecute.stepWaitAndRegisterOutgoingResult))

		smExecute.stepSendOutgoing(execCtx)
	}

	mc.Finish()
}
