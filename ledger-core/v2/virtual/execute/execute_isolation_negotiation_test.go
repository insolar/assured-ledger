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

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/runner"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils"
)

func Test_Execute_stepIsolationNegotiation(t *testing.T) {
	t.Run("constuctor", func(t *testing.T) {
		var (
			ctx = inslogger.TestContext(t)
			mc  = minimock.NewController(t)

			pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
			pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
			catalog         = object.NewCatalogMock(mc)
			smObjectID      = gen.IDWithPulse(pd.PulseNumber)
			smGlobalRef     = reference.NewSelf(smObjectID)
			smObject        = object.NewStateMachineObject(smGlobalRef)
			sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

			callFlags payload.CallRequestFlags
		)

		callFlags.SetInterference(contract.CallTolerable)
		callFlags.SetState(contract.CallDirty)

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
			objectCatalog:     catalog,
			pulseSlot:         &pulseSlot,
			objectSharedState: object.SharedStateAccessor{SharedDataLink: sharedStateData},
		}

		stepChecker := testutils.NewSMStepChecker()
		{
			exec := SMExecute{}
			stepChecker.AddStep(exec.stepIsolationNegotiation)
			stepChecker.AddStep(exec.stepTakeLock)
		}

		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t)).
			UseSharedMock.Set(CallSharedDataAccessor).
			AcquireForThisStepMock.Return(true)

		smExecute.prepareExecution(ctx)
		smExecute.stepWaitObjectReady(execCtx)

		smExecute.stepIsolationNegotiation(execCtx)

		assert.Equal(t, contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}, smExecute.execution.Isolation)

		mc.Finish()
	})

	t.Run("bad constuctor", func(t *testing.T) {
		var (
			ctx = inslogger.TestContext(t)
			mc  = minimock.NewController(t)

			pd              = pulse.NewFirstPulsarData(10, longbits.Bits256{})
			pulseSlot       = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
			catalog         = object.NewCatalogMock(mc)
			smObjectID      = gen.IDWithPulse(pd.PulseNumber)
			smGlobalRef     = reference.NewSelf(smObjectID)
			smObject        = object.NewStateMachineObject(smGlobalRef)
			sharedStateData = smachine.NewUnboundSharedData(&smObject.SharedState)

			callFlags payload.CallRequestFlags
		)

		callFlags.SetInterference(contract.CallIntolerable)
		callFlags.SetState(contract.CallDirty)

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
			objectCatalog:     catalog,
			pulseSlot:         &pulseSlot,
			objectSharedState: object.SharedStateAccessor{SharedDataLink: sharedStateData},
		}

		stepChecker := testutils.NewSMStepChecker()
		{
			exec := SMExecute{}
			stepChecker.AddStep(exec.stepIsolationNegotiation)
			stepChecker.AddStep(exec.stepTakeLock)
		}

		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t)).
			UseSharedMock.Set(CallSharedDataAccessor).
			AcquireForThisStepMock.Return(true)

		// expected SM stop with Error
		execCtx.ErrorMock.Set(func(e1 error) (s1 smachine.StateUpdate) {
			require.Error(t, e1)
			return smachine.StateUpdate{}
		})

		smExecute.prepareExecution(ctx)

		smExecute.stepWaitObjectReady(execCtx)
		smExecute.stepIsolationNegotiation(execCtx)

		mc.Finish()
	})

	t.Run("method immutable", func(t *testing.T) {
		var (
			ctx = inslogger.TestContext(t)
			mc  = minimock.NewController(t)

			pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
			pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
			catalog     = object.NewCatalogMock(mc)
			smObjectID  = gen.IDWithPulse(pd.PulseNumber)
			smGlobalRef = reference.NewSelf(smObjectID)
			smObject    = object.NewStateMachineObject(smGlobalRef)
			callFlags   payload.CallRequestFlags
		)

		smObject.SharedState.SetDescriptor(descriptor.NewObject(smGlobalRef, smObjectID, testwallet.GetPrototype(), nil, reference.Global{}))
		sharedStateData := smachine.NewUnboundSharedData(&smObject.SharedState)

		runnerService := runner.NewService()
		require.NoError(t, runnerService.Init())

		callFlags.SetInterference(contract.CallIntolerable)
		callFlags.SetState(contract.CallValidated)

		smExecute := SMExecute{
			Payload: &payload.VCallRequest{
				Polymorph:           uint32(payload.TypeVCallRequest),
				CallType:            payload.CTMethod,
				CallFlags:           callFlags,
				CallSiteDeclaration: testwallet.GetPrototype(),
				CallSiteMethod:      "GetBalance",
				CallOutgoing:        smObjectID,
				Arguments:           insolar.MustSerialize([]interface{}{}),
			},
			objectCatalog:     catalog,
			pulseSlot:         &pulseSlot,
			objectSharedState: object.SharedStateAccessor{SharedDataLink: sharedStateData},

			methodIsolation: &contract.MethodIsolation{
				Interference: contract.CallIntolerable,
				State:        contract.CallValidated,
			},
		}

		stepChecker := testutils.NewSMStepChecker()
		{
			exec := SMExecute{}
			stepChecker.AddStep(exec.stepIsolationNegotiation)
			stepChecker.AddStep(exec.stepTakeLock)
		}

		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t)).
			UseSharedMock.Set(CallSharedDataAccessor).
			AcquireForThisStepMock.Return(true)

		execCtx.ErrorMock.Inspect(func(e1 error) {
			require.NoError(t, e1)
		})

		smExecute.prepareExecution(ctx)

		smExecute.stepWaitObjectReady(execCtx)
		smExecute.stepIsolationNegotiation(execCtx)

		assert.Equal(t, contract.MethodIsolation{
			Interference: contract.CallIntolerable,
			State:        contract.CallValidated,
		}, smExecute.execution.Isolation)

		mc.Finish()
	})

	t.Run("method mutable", func(t *testing.T) {
		var (
			ctx = inslogger.TestContext(t)
			mc  = minimock.NewController(t)

			pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
			pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
			catalog     = object.NewCatalogMock(mc)
			smObjectID  = gen.IDWithPulse(pd.PulseNumber)
			smGlobalRef = reference.NewSelf(smObjectID)
			smObject    = object.NewStateMachineObject(smGlobalRef)
			callFlags   payload.CallRequestFlags
		)

		smObject.SharedState.SetDescriptor(descriptor.NewObject(smGlobalRef, smObjectID, testwallet.GetPrototype(), nil, reference.Global{}))
		sharedStateData := smachine.NewUnboundSharedData(&smObject.SharedState)

		runnerService := runner.NewService()
		require.NoError(t, runnerService.Init())

		callFlags.SetInterference(contract.CallTolerable)
		callFlags.SetState(contract.CallDirty)

		smExecute := SMExecute{
			Payload: &payload.VCallRequest{
				Polymorph:           uint32(payload.TypeVCallRequest),
				CallType:            payload.CTMethod,
				CallFlags:           callFlags,
				CallSiteDeclaration: testwallet.GetPrototype(),
				CallSiteMethod:      "Transfer",
				CallOutgoing:        smObjectID,
				Arguments:           insolar.MustSerialize([]interface{}{}),
			},
			objectCatalog:     catalog,
			pulseSlot:         &pulseSlot,
			objectSharedState: object.SharedStateAccessor{SharedDataLink: sharedStateData},

			methodIsolation: &contract.MethodIsolation{
				Interference: contract.CallTolerable,
				State:        contract.CallDirty,
			},
		}

		stepChecker := testutils.NewSMStepChecker()
		{
			exec := SMExecute{}
			stepChecker.AddStep(exec.stepIsolationNegotiation)
			stepChecker.AddStep(exec.stepTakeLock)
		}

		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t)).
			UseSharedMock.Set(CallSharedDataAccessor).
			AcquireForThisStepMock.Return(true)

		execCtx.ErrorMock.Inspect(func(e1 error) {
			require.NoError(t, e1)
		})

		smExecute.prepareExecution(ctx)

		smExecute.stepWaitObjectReady(execCtx)
		smExecute.stepIsolationNegotiation(execCtx)

		assert.Equal(t, contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}, smExecute.execution.Isolation)

		mc.Finish()
	})

	t.Run("mixed tolerance", func(t *testing.T) {
		var (
			ctx = inslogger.TestContext(t)
			mc  = minimock.NewController(t)

			pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
			pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
			catalog     = object.NewCatalogMock(mc)
			smObjectID  = gen.IDWithPulse(pd.PulseNumber)
			smGlobalRef = reference.NewSelf(smObjectID)
			smObject    = object.NewStateMachineObject(smGlobalRef)
			callFlags   payload.CallRequestFlags
		)

		smObject.SharedState.SetDescriptor(descriptor.NewObject(smGlobalRef, smObjectID, testwallet.GetPrototype(), nil, reference.Global{}))
		sharedStateData := smachine.NewUnboundSharedData(&smObject.SharedState)

		runnerService := runner.NewService()
		require.NoError(t, runnerService.Init())

		callFlags.SetInterference(contract.CallTolerable)
		callFlags.SetState(contract.CallDirty)

		smExecute := SMExecute{
			Payload: &payload.VCallRequest{
				Polymorph:           uint32(payload.TypeVCallRequest),
				CallType:            payload.CTMethod,
				CallFlags:           callFlags,
				CallSiteDeclaration: testwallet.GetPrototype(),
				CallSiteMethod:      "Transfer",
				CallOutgoing:        smObjectID,
				Arguments:           insolar.MustSerialize([]interface{}{}),
			},
			objectCatalog:     catalog,
			pulseSlot:         &pulseSlot,
			objectSharedState: object.SharedStateAccessor{SharedDataLink: sharedStateData},

			methodIsolation: &contract.MethodIsolation{
				Interference: contract.CallIntolerable,
				State:        contract.CallDirty,
			},
		}

		stepChecker := testutils.NewSMStepChecker()
		{
			exec := SMExecute{}
			stepChecker.AddStep(exec.stepIsolationNegotiation)
			stepChecker.AddStep(exec.stepTakeLock)
		}

		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t)).
			UseSharedMock.Set(CallSharedDataAccessor).
			AcquireForThisStepMock.Return(true)

		execCtx.ErrorMock.Inspect(func(e1 error) {
			require.NoError(t, e1)
		})

		smExecute.prepareExecution(ctx)

		smExecute.stepWaitObjectReady(execCtx)
		smExecute.stepIsolationNegotiation(execCtx)

		assert.Equal(t, contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}, smExecute.execution.Isolation)

		mc.Finish()
	})

	t.Run("bad tolerance", func(t *testing.T) {
		var (
			ctx = inslogger.TestContext(t)
			mc  = minimock.NewController(t)

			pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
			pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
			catalog     = object.NewCatalogMock(mc)
			smObjectID  = gen.IDWithPulse(pd.PulseNumber)
			smGlobalRef = reference.NewSelf(smObjectID)
			smObject    = object.NewStateMachineObject(smGlobalRef)
			callFlags   payload.CallRequestFlags
		)

		smObject.SharedState.SetDescriptor(descriptor.NewObject(smGlobalRef, smObjectID, testwallet.GetPrototype(), nil, reference.Global{}))
		sharedStateData := smachine.NewUnboundSharedData(&smObject.SharedState)

		runnerService := runner.NewService()
		require.NoError(t, runnerService.Init())

		callFlags.SetInterference(contract.CallIntolerable)
		callFlags.SetState(contract.CallDirty)

		smExecute := SMExecute{
			Payload: &payload.VCallRequest{
				Polymorph:           uint32(payload.TypeVCallRequest),
				CallType:            payload.CTMethod,
				CallFlags:           callFlags,
				CallSiteDeclaration: testwallet.GetPrototype(),
				CallSiteMethod:      "Transfer",
				CallOutgoing:        smObjectID,
				Arguments:           insolar.MustSerialize([]interface{}{}),
			},
			objectCatalog:     catalog,
			pulseSlot:         &pulseSlot,
			objectSharedState: object.SharedStateAccessor{SharedDataLink: sharedStateData},

			methodIsolation: &contract.MethodIsolation{
				Interference: contract.CallTolerable,
				State:        contract.CallDirty,
			},
		}

		stepChecker := testutils.NewSMStepChecker()
		{
			exec := SMExecute{}
			stepChecker.AddStep(exec.stepIsolationNegotiation)
			stepChecker.AddStep(exec.stepTakeLock)
		}

		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t)).
			UseSharedMock.Set(CallSharedDataAccessor).
			AcquireForThisStepMock.Return(true)

		// expected SM stop with Error
		execCtx.ErrorMock.Set(func(e1 error) (s1 smachine.StateUpdate) {
			require.Error(t, e1)
			return smachine.StateUpdate{}
		})

		smExecute.prepareExecution(ctx)

		smExecute.stepWaitObjectReady(execCtx)
		smExecute.stepIsolationNegotiation(execCtx)

		mc.Finish()
	})

	t.Run("mixed state", func(t *testing.T) {
		var (
			ctx = inslogger.TestContext(t)
			mc  = minimock.NewController(t)

			pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
			pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
			catalog     = object.NewCatalogMock(mc)
			smObjectID  = gen.IDWithPulse(pd.PulseNumber)
			smGlobalRef = reference.NewSelf(smObjectID)
			smObject    = object.NewStateMachineObject(smGlobalRef)
			callFlags   payload.CallRequestFlags
		)

		smObject.SharedState.SetDescriptor(descriptor.NewObject(smGlobalRef, smObjectID, testwallet.GetPrototype(), nil, reference.Global{}))
		sharedStateData := smachine.NewUnboundSharedData(&smObject.SharedState)

		runnerService := runner.NewService()
		require.NoError(t, runnerService.Init())

		callFlags.SetInterference(contract.CallTolerable)
		callFlags.SetState(contract.CallDirty)

		smExecute := SMExecute{
			Payload: &payload.VCallRequest{
				Polymorph:           uint32(payload.TypeVCallRequest),
				CallType:            payload.CTMethod,
				CallFlags:           callFlags,
				CallSiteDeclaration: testwallet.GetPrototype(),
				CallSiteMethod:      "Transfer",
				CallOutgoing:        smObjectID,
				Arguments:           insolar.MustSerialize([]interface{}{}),
			},
			objectCatalog:     catalog,
			pulseSlot:         &pulseSlot,
			objectSharedState: object.SharedStateAccessor{SharedDataLink: sharedStateData},

			methodIsolation: &contract.MethodIsolation{
				Interference: contract.CallTolerable,
				State:        contract.CallValidated,
			},
		}

		stepChecker := testutils.NewSMStepChecker()
		{
			exec := SMExecute{}
			stepChecker.AddStep(exec.stepIsolationNegotiation)
			stepChecker.AddStep(exec.stepTakeLock)
		}

		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t)).
			UseSharedMock.Set(CallSharedDataAccessor).
			AcquireForThisStepMock.Return(true)

		execCtx.ErrorMock.Inspect(func(e1 error) {
			require.NoError(t, e1)
		})

		smExecute.prepareExecution(ctx)

		smExecute.stepWaitObjectReady(execCtx)
		smExecute.stepIsolationNegotiation(execCtx)

		assert.Equal(t, contract.MethodIsolation{
			Interference: contract.CallTolerable,
			State:        contract.CallDirty,
		}, smExecute.execution.Isolation)

		mc.Finish()
	})

	t.Run("bad state", func(t *testing.T) {
		var (
			ctx = inslogger.TestContext(t)
			mc  = minimock.NewController(t)

			pd          = pulse.NewFirstPulsarData(10, longbits.Bits256{})
			pulseSlot   = conveyor.NewPresentPulseSlot(nil, pd.AsRange())
			catalog     = object.NewCatalogMock(mc)
			smObjectID  = gen.IDWithPulse(pd.PulseNumber)
			smGlobalRef = reference.NewSelf(smObjectID)
			smObject    = object.NewStateMachineObject(smGlobalRef)
			callFlags   payload.CallRequestFlags
		)

		smObject.SharedState.SetDescriptor(descriptor.NewObject(smGlobalRef, smObjectID, testwallet.GetPrototype(), nil, reference.Global{}))
		sharedStateData := smachine.NewUnboundSharedData(&smObject.SharedState)

		runnerService := runner.NewService()
		require.NoError(t, runnerService.Init())

		callFlags.SetInterference(contract.CallTolerable)
		callFlags.SetState(contract.CallValidated)

		smExecute := SMExecute{
			Payload: &payload.VCallRequest{
				Polymorph:           uint32(payload.TypeVCallRequest),
				CallType:            payload.CTMethod,
				CallFlags:           callFlags,
				CallSiteDeclaration: testwallet.GetPrototype(),
				CallSiteMethod:      "Transfer",
				CallOutgoing:        smObjectID,
				Arguments:           insolar.MustSerialize([]interface{}{}),
			},
			objectCatalog:     catalog,
			pulseSlot:         &pulseSlot,
			objectSharedState: object.SharedStateAccessor{SharedDataLink: sharedStateData},

			methodIsolation: &contract.MethodIsolation{
				Interference: contract.CallTolerable,
				State:        contract.CallDirty,
			},
		}

		stepChecker := testutils.NewSMStepChecker()
		{
			exec := SMExecute{}
			stepChecker.AddStep(exec.stepIsolationNegotiation)
			stepChecker.AddStep(exec.stepTakeLock)
		}

		execCtx := smachine.NewExecutionContextMock(mc).
			JumpMock.Set(stepChecker.CheckJumpW(t)).
			UseSharedMock.Set(CallSharedDataAccessor).
			AcquireForThisStepMock.Return(true)

		// expected SM stop with Error
		execCtx.ErrorMock.Set(func(e1 error) (s1 smachine.StateUpdate) {
			require.Error(t, e1)
			return smachine.StateUpdate{}
		})

		smExecute.prepareExecution(ctx)

		smExecute.stepWaitObjectReady(execCtx)
		smExecute.stepIsolationNegotiation(execCtx)

		mc.Finish()
	})
}
