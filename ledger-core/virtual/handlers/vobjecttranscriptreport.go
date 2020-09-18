// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

//go:generate sm-uml-gen -f $GOFILE

package handlers

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	memoryCacheAdapter "github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/adapter"
	"github.com/insolar/assured-ledger/ledger-core/virtual/memorycache/statemachine"
)

type CachedMemoryReportAwaitKey struct {
	State reference.Global
}

type SMVObjectTranscriptReport struct {
	// input arguments
	Meta    *rms.Meta
	Payload *rms.VObjectTranscriptReport

	// deps
	runner        runner.ServiceAdapter
	pulseSlot     *conveyor.PulseSlot
	messageSender messageSenderAdapter.MessageSender
	memoryCache   memoryCacheAdapter.MemoryCache

	// unboxed from message, often used
	object  reference.Global
	entries []rms.Any

	objState        reference.Global
	objDesc         descriptor.Object
	entryIndex      int
	incomingRequest *rms.VCallRequest
	outgoingRequest *rms.VCallRequest
	outgoingResult  *rms.VCallResult

	validatedState reference.Global

	execution         execution.Context
	executionNewState *execution.Update
	run               runner.RunState
}

/* -------- Declaration ------------- */

var dSMVObjectTranscriptReportInstance smachine.StateMachineDeclaration = &dSMVObjectTranscriptReport{}

type dSMVObjectTranscriptReport struct {
	smachine.StateMachineDeclTemplate
}

func (*dSMVObjectTranscriptReport) InjectDependencies(sm smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	s := sm.(*SMVObjectTranscriptReport)

	injector.MustInject(&s.runner)
	injector.MustInject(&s.pulseSlot)
	injector.MustInject(&s.memoryCache)
	injector.MustInject(&s.messageSender)
}

func (*dSMVObjectTranscriptReport) GetInitStateFor(sm smachine.StateMachine) smachine.InitFunc {
	s := sm.(*SMVObjectTranscriptReport)
	return s.Init
}

/* -------- Instance ------------- */

func (s *SMVObjectTranscriptReport) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return dSMVObjectTranscriptReportInstance
}

func (s *SMVObjectTranscriptReport) Init(ctx smachine.InitializationContext) smachine.StateUpdate {
	s.object = s.Payload.Object.GetValue()
	s.entries = s.Payload.ObjectTranscript.GetEntries()

	return ctx.Jump(s.stepProcess)
}

func (s *SMVObjectTranscriptReport) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	entry := s.peekCurrentEntry()

	switch tEntry := entry.(type) {
	case *rms.VObjectTranscriptReport_TranscriptEntryIncomingRequest:
		s.incomingRequest = &tEntry.Request
		s.objState = tEntry.ObjectMemory.GetValue()
		if s.objState.IsEmpty() {
			return ctx.Jump(s.stepIncomingRequest)
		}
		return ctx.Jump(s.stepGetMemory)
	default:
		// TODO: no idea how deal here with this
		panic(throw.IllegalValue())
	}

	return ctx.Stop()
}

func (s *SMVObjectTranscriptReport) stepGetMemory(ctx smachine.ExecutionContext) smachine.StateUpdate {
	subSM := &statemachine.SMGetCachedMemory{
		Object: s.object, State: s.objState.GetLocal(),
	}
	return ctx.CallSubroutine(subSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
		if subSM.Result == nil {
			panic(throw.IllegalState())
		}
		s.objDesc = subSM.Result
		return ctx.Jump(s.stepIncomingRequest)
	})
}

func (s *SMVObjectTranscriptReport) stepIncomingRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.prepareExecution(ctx.GetContext())
	return ctx.Jump(s.stepExecuteStart)
}

func (s *SMVObjectTranscriptReport) stepExecuteStart(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.run = nil
	return s.runner.PrepareExecutionStart(ctx, s.execution, func(state runner.RunState) {
		if state == nil {
			panic(throw.IllegalValue())
		}
		s.run = state

		s.executionNewState = state.GetResult()
		if s.executionNewState == nil {
			panic(throw.IllegalValue())
		}
	}).DelayedStart().ThenJump(s.stepWaitExecutionResult)
}

func (s *SMVObjectTranscriptReport) stepExecuteContinue(ctx smachine.ExecutionContext) smachine.StateUpdate {
	outgoingResult := s.outgoingResult
	switch s.executionNewState.Outgoing.(type) {
	case execution.CallConstructor, execution.CallMethod:
		if outgoingResult == nil {
			panic(throw.IllegalValue())
		}
	}

	s.executionNewState = nil

	executionResult := requestresult.NewOutgoingExecutionResult(outgoingResult.ReturnArguments.GetBytes(), nil)
	return s.runner.PrepareExecutionContinue(ctx, s.run, executionResult, func() {
		if s.run == nil {
			panic(throw.IllegalState())
		}

		s.executionNewState = s.run.GetResult()
		if s.executionNewState == nil {
			panic(throw.IllegalState())
		}
	}).DelayedStart().ThenJump(s.stepWaitExecutionResult)
}

func (s *SMVObjectTranscriptReport) stepWaitExecutionResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.executionNewState == nil {
		return ctx.Sleep().ThenRepeat()
	}
	return ctx.Jump(s.stepExecuteDecideNextStep)
}

func (s *SMVObjectTranscriptReport) stepExecuteDecideNextStep(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.executionNewState == nil {
		panic(throw.IllegalState())
	}

	newState := s.executionNewState

	switch newState.Type {
	case execution.Done:
	case execution.OutgoingCall:
		return ctx.Jump(s.stepExecuteOutgoing)
	case execution.Error, execution.Abort:
		panic(throw.NotImplemented())
	default:
		panic(throw.IllegalValue())
	}

	return ctx.Jump(s.stepExecuteFinish)
}

func (s *SMVObjectTranscriptReport) stepExecuteOutgoing(ctx smachine.ExecutionContext) smachine.StateUpdate {
	pulseNumber := s.execution.Pulse.PulseNumber

	switch outgoing := s.executionNewState.Outgoing.(type) {
	case execution.Deactivate:
		panic(throw.NotImplemented())
	case execution.CallConstructor:
		s.outgoingRequest = outgoing.ConstructVCallRequest(s.execution)
		newOutgoing := reference.NewRecordOf(s.outgoingRequest.Caller.GetValue(), gen.UniqueLocalRefWithPulse(pulseNumber))
		s.outgoingRequest.CallOutgoing.Set(newOutgoing)
	case execution.CallMethod:
		s.outgoingRequest = outgoing.ConstructVCallRequest(s.execution)
		newOutgoing := reference.NewRecordOf(s.outgoingRequest.Caller.GetValue(), gen.UniqueLocalRefWithPulse(pulseNumber))
		s.outgoingRequest.CallOutgoing.Set(newOutgoing)
	default:
		panic(throw.IllegalValue())
	}

	entry := s.peekNextEntry()
	expectedRequest, ok := entry.(*rms.VObjectTranscriptReport_TranscriptEntryOutgoingRequest)
	if !ok {
		panic(throw.NotImplemented())
	}
	equal := s.outgoingRequest.CallOutgoing.Equal(&expectedRequest.Request)
	if !equal {
		// todo: fixme: validation failed, CallOutgoing is random for now
		//panic(throw.NotImplemented())
	}
	s.entryIndex++

	entry = s.peekNextEntry()
	outgoingResult, ok := entry.(*rms.VObjectTranscriptReport_TranscriptEntryOutgoingResult)
	if !ok {
		panic(throw.NotImplemented())
	}
	s.outgoingResult = &outgoingResult.CallResult
	s.entryIndex++

	return ctx.Jump(s.stepExecuteContinue)
}

func (s *SMVObjectTranscriptReport) stepExecuteFinish(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var newDesc descriptor.Object

	switch s.executionNewState.Result.Type() {
	case requestresult.SideEffectNone:
	case requestresult.SideEffectActivate:
		class, memory := s.executionNewState.Result.Activate()
		newDesc = s.makeNewDescriptor(class, memory, false)
	case requestresult.SideEffectAmend:
		class, memory := s.executionNewState.Result.Amend()
		newDesc = s.makeNewDescriptor(class, memory, false)
	case requestresult.SideEffectDeactivate:
		class, memory := s.executionNewState.Result.Deactivate()
		newDesc = s.makeNewDescriptor(class, memory, true)
	default:
		panic(throw.IllegalValue())
	}

	entry := s.peekNextEntry()
	expected := entry.(*rms.VObjectTranscriptReport_TranscriptEntryIncomingResult)

	// fixme: stateid vs stateref
	stateRef := reference.NewRecordOf(newDesc.HeadRef(), newDesc.StateID())
	equal := stateRef.Equal(expected.ObjectState.GetValue())
	if !equal {
		return ctx.Jump(s.stepValidationFailed)
	}

	s.validatedState = stateRef
	return ctx.Jump(s.stepAdvanceToNextRequest)
}

func (s *SMVObjectTranscriptReport) stepValidationFailed(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// todo: validation failed
	panic(throw.NotImplemented())
}

func (s *SMVObjectTranscriptReport) stepAdvanceToNextRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.entryIndex++
	if s.entryIndex >= len(s.entries)-1 {
		if !s.validatedState.IsEmpty() {
			return ctx.Jump(s.stepSendValidationReport)
		} else {
			panic(throw.NotImplemented())
		}
	}
	return ctx.Stop()
}

func (s *SMVObjectTranscriptReport) stepSendValidationReport(ctx smachine.ExecutionContext) smachine.StateUpdate {
	msg := rms.VObjectValidationReport{
		Object:    rms.NewReference(s.object),
		In:        s.pulseSlot.PulseNumber(),
		Validated: rms.NewReference(s.validatedState),
	}
	s.messageSender.PrepareAsync(ctx, func(goCtx context.Context, svc messagesender.Service) smachine.AsyncResultFunc {
		err := svc.SendRole(goCtx, &msg, affinity.DynamicRoleVirtualExecutor, s.object, s.pulseSlot.PulseNumber())
		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				ctx.Log().Error("failed to send message", err)
			}
		}
	}).WithoutAutoWakeUp().Start()

	return ctx.Stop()
}

func (s *SMVObjectTranscriptReport) peekEntry(index int) rmsreg.GoGoSerializable {
	return s.entries[index].Get()
}

func (s *SMVObjectTranscriptReport) peekCurrentEntry() rmsreg.GoGoSerializable {
	return s.peekEntry(s.entryIndex)
}

func (s *SMVObjectTranscriptReport) peekNextEntry() rmsreg.GoGoSerializable {
	return s.peekEntry(s.entryIndex + 1)
}

// FIXME: copy&paste from execute.go, also many things around c&p, we should re-use code
func (s *SMVObjectTranscriptReport) makeNewDescriptor(
	class reference.Global,
	memory []byte,
	deactivated bool,
) descriptor.Object {
	var prevStateIDBytes []byte
	objDescriptor := s.objDesc
	if objDescriptor != nil {
		prevStateIDBytes = objDescriptor.StateID().AsBytes()
	}

	objectRefBytes := s.object.AsBytes()
	stateHash := append(memory, objectRefBytes...)
	stateHash = append(stateHash, prevStateIDBytes...)

	stateID := execute.NewStateID(s.pulseSlot.PulseData().GetPulseNumber(), stateHash)
	return descriptor.NewObject(
		s.object,
		stateID,
		class,
		memory,
		deactivated,
	)
}

func (s *SMVObjectTranscriptReport) prepareExecution(ctx context.Context) {
	s.execution = execute.ExecContextFromRequest(s.incomingRequest)

	s.execution.Context = ctx
	s.execution.Pulse = s.pulseSlot.PulseData()
}
