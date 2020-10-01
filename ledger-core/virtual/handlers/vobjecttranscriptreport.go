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
	object   reference.Global
	entries  []rms.Any
	pendings []rms.Transcript

	objState        reference.Global
	objDesc         descriptor.Object
	reasonRef       rms.Reference
	entryIndex      int
	startIndex      int
	counter         int
	incomingRequest *rms.VCallRequest
	outgoingRequest *rms.VCallRequest
	outgoingResult  *rms.VCallResult
	withPendings    bool

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
	s.pendings = s.Payload.PendingTranscripts

	if len(s.pendings) != 0 {
		s.withPendings = true
		s.entries = make([]rms.Any, 0, len(s.pendings)*2)
		for _, transcript := range s.pendings {
			s.entries = append(s.entries, transcript.Entries...)
		}
	}

	s.entries = append(s.entries, s.Payload.ObjectTranscript.GetEntries()...)

	return ctx.Jump(s.stepProcess)
}

func (s *SMVObjectTranscriptReport) stepProcess(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if len(s.entries) == 0 {
		panic(throw.Impossible())
	}
	if ok := len(s.entries) % 2; ok != 0 {
		panic(throw.Impossible())
	}

	entry := s.peekEntry(s.startIndex)
	if entry == nil {
		ctx.Log().Warn("validation failed: can't find TranscriptEntryIncomingRequest")
		return ctx.Jump(s.stepValidationFailed)
	}
	s.entryIndex = s.startIndex
	s.counter++

	switch tEntry := entry.(type) {
	case *rms.Transcript_TranscriptEntryIncomingRequest:
		s.incomingRequest = &tEntry.Request
		s.objState = tEntry.ObjectMemory.GetValue()
		s.reasonRef = s.incomingRequest.CallOutgoing
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
		Object: s.object, State: s.objState.GetLocal(), Class: s.Payload.Class.GetValue(),
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
	s.executionNewState = nil
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
		ctx.Log().Error("execution failed", newState.Error)
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

	entry := s.findNextEntry()
	if entry == nil {
		ctx.Log().Warn("validation failed: can't find TranscriptEntryOutgoingRequest")
		return ctx.Jump(s.stepValidationFailed)
	}
	s.counter++
	expectedRequest, ok := entry.(*rms.Transcript_TranscriptEntryOutgoingRequest)
	if !ok {
		ctx.Log().Warn("validation failed: failed to convert GoGoSerializable object to Transcript_TranscriptEntryOutgoingRequest")
		return ctx.Jump(s.stepValidationFailed)
	}
	equal := s.outgoingRequest.CallOutgoing.Equal(&expectedRequest.Request)
	if !equal {
		// todo: fixme: validation failed, CallOutgoing is random for now
		//panic(throw.NotImplemented())
	}

	entry = s.findNextEntry()
	if entry == nil {
		ctx.Log().Warn("validation failed: can't find TranscriptEntryOutgoingResult")
		return ctx.Jump(s.stepValidationFailed)
	}
	s.counter++
	outgoingResult, ok := entry.(*rms.Transcript_TranscriptEntryOutgoingResult)
	if !ok {
		ctx.Log().Warn("validation failed: failed to convert GoGoSerializable object to Transcript_TranscriptEntryOutgoingResult")
		return ctx.Jump(s.stepValidationFailed)
	}
	s.outgoingResult = &outgoingResult.CallResult

	return ctx.Jump(s.stepExecuteContinue)
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

func (s *SMVObjectTranscriptReport) stepExecuteFinish(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var (
		newDesc      descriptor.Object
		noSideEffect bool
	)

	switch s.executionNewState.Result.Type() {
	case requestresult.SideEffectNone:
		noSideEffect = true
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

	entry := s.findNextEntry()
	if entry == nil {
		ctx.Log().Warn("validation failed: can't find TranscriptEntryIncomingResult")
		return ctx.Jump(s.stepValidationFailed)
	}
	s.counter++
	callResult, ok := entry.(*rms.Transcript_TranscriptEntryIncomingResult)
	if !ok {
		ctx.Log().Warn("validation failed: failed to convert GoGoSerializable object to Transcript_TranscriptEntryIncomingResult")
		return ctx.Jump(s.stepValidationFailed)
	}

	// fixme: we still should compare if there is no side effect
	if !noSideEffect {
		// it's pending, check headRef base and stateHash without pulse
		// fixme: hack for validation, need to rethink
		// fixme: stateid vs stateref
		headRef := callResult.ObjectState.GetValue().GetBase()
		if !headRef.Equal(s.object.GetBase()) {
			ctx.Log().Warn("pending validation failed: wrong headRef base")
			return ctx.Jump(s.stepValidationFailed)
		}
		stateHash := callResult.ObjectState.GetValue().GetLocal().GetHash()
		if stateHash.Compare(newDesc.StateID().GetHash()) != 0 {
			ctx.Log().Warn("pending validation failed: wrong stateHash")
			return ctx.Jump(s.stepValidationFailed)
		}
		s.validatedState = callResult.ObjectState.GetValue()
	} else if callResult.ObjectState.GetValue().GetLocal().Equal(s.objDesc.StateID()) {
		// FIXME: we don't really change value of validateState of the object, we shouldn't
		// send message at the end when all requests are intollerable
		// or don't change memory
		s.validatedState = callResult.ObjectState.GetValue()
	} else {
		ctx.Log().Warn("pending validation failed: wrong stateHash")
		return ctx.Jump(s.stepValidationFailed)
	}

	return ctx.Jump(s.stepAdvanceToNextRequest)
}

func (s *SMVObjectTranscriptReport) stepValidationFailed(ctx smachine.ExecutionContext) smachine.StateUpdate {
	// todo: validation failed
	panic(throw.NotImplemented())
}

func (s *SMVObjectTranscriptReport) stepAdvanceToNextRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.counter == len(s.entries) {
		// everything is checked
		if s.validatedState.IsEmpty() {
			panic(throw.NotImplemented())
		}

		return ctx.Jump(s.stepSendValidationReport)
	}

	s.startIndex++
	s.entryIndex = 0

	for ind := s.startIndex; ind < len(s.entries); ind++ {
		entry := s.peekEntry(ind)
		if _, ok := entry.(*rms.Transcript_TranscriptEntryIncomingRequest); ok {
			s.startIndex = ind
			break
		}
	}

	return ctx.Jump(s.stepProcess)
}

func (s *SMVObjectTranscriptReport) stepSendValidationReport(ctx smachine.ExecutionContext) smachine.StateUpdate {
	msg := rms.VObjectValidationReport{
		Object:    rms.NewReference(s.object),
		Class:     rms.NewReference(s.Payload.Class.GetValue()),
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

func (s *SMVObjectTranscriptReport) findNextEntry() rmsreg.GoGoSerializable {
	for ind := s.entryIndex + 1; ind < len(s.entries); ind++ {
		entry := s.peekEntry(ind)
		switch entry.(type) {
		case *rms.Transcript_TranscriptEntryIncomingResult:
			expected := entry.(*rms.Transcript_TranscriptEntryIncomingResult)
			if expected.Reason.Equal(&s.reasonRef) {
				s.entryIndex = ind
				return entry
			}
		case *rms.Transcript_TranscriptEntryOutgoingRequest:
			expected := entry.(*rms.Transcript_TranscriptEntryOutgoingRequest)
			if expected.Reason.Equal(&s.reasonRef) {
				s.entryIndex = ind
				return entry
			}
		case *rms.Transcript_TranscriptEntryOutgoingResult:
			expected := entry.(*rms.Transcript_TranscriptEntryOutgoingResult)
			if expected.Reason.Equal(&s.reasonRef) {
				s.entryIndex = ind
				return entry
			}
		default:
			// do nothing
		}
	}
	return nil
}

func (s *SMVObjectTranscriptReport) makeNewDescriptor(
	class reference.Global,
	memory []byte,
	deactivated bool,
) descriptor.Object {
	return execute.MakeDescriptor(
		s.objDesc,
		s.object,
		class, memory, deactivated,
		// FIXME: incorrect pulse
		s.pulseSlot.PulseData().GetPulseNumber(),
	)
}

func (s *SMVObjectTranscriptReport) prepareExecution(ctx context.Context) {
	s.execution = execute.ExecContextFromRequest(s.incomingRequest)

	s.execution.Context = ctx
	s.execution.Pulse = s.pulseSlot.PulseData()
	s.execution.ObjectDescriptor = s.objDesc
}
