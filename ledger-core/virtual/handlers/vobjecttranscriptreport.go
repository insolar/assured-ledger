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
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/runner"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/execute"
	"github.com/insolar/assured-ledger/ledger-core/virtual/lmn"
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

	objState          reference.Global
	objDesc           descriptor.Object
	reasonRef         rms.Reference
	entryIndex        int
	startIndex        int
	counter           int
	incomingRequest   *rms.VCallRequest
	outgoingRequest   *rms.VCallRequest
	outgoingResult    *rms.VCallResult
	outgoingResultRef reference.Global
	incomingRecordRef rms.Reference
	withPendings      bool

	// for lmn
	lmnLastFilamentRef    reference.Global
	lmnLastLifelineRef    reference.Global
	lmnIncomingRequestRef reference.Global
	incomingRegistered    bool
	incomingValidated     bool

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
		s.incomingRecordRef = tEntry.Incoming
		if s.objState.IsEmpty() {
			return ctx.Jump(s.stepPrepareExecutionContext)
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
		Object: s.object, State: s.objState,
	}
	return ctx.CallSubroutine(subSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
		if subSM.Result == nil {
			panic(throw.IllegalState())
		}
		s.objDesc = subSM.Result
		s.lmnLastLifelineRef = s.objDesc.State()
		return ctx.Jump(s.stepPrepareExecutionContext)
	})
}

func (s *SMVObjectTranscriptReport) stepPrepareExecutionContext(ctx smachine.ExecutionContext) smachine.StateUpdate {
	s.prepareExecution(ctx.GetContext())
	return ctx.Jump(s.stepInboundRecord)
}

func (s *SMVObjectTranscriptReport) stepInboundRecord(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if s.incomingRequest.CallType == rms.CallTypeConstructor {
		return ctx.Jump(s.stepConstructorRecord)
	}
	// for VCallMethod InboundRecord is calculated with OutboundRecord or InboundResponse
	return ctx.Jump(s.stepExecuteStart)
}

func (s *SMVObjectTranscriptReport) stepConstructorRecord(ctx smachine.ExecutionContext) smachine.StateUpdate {
	subroutineSM := s.constructSubSMRegister(RegisterLifeLine)
	subroutineSM.Incoming = s.incomingRequest

	return ctx.CallSubroutine(&subroutineSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
		if ctx.GetError() != nil {
			// TODO: we should understand here what's happened, but for now we'll drop request execution here
			return ctx.Error(ctx.GetError())
		}

		if subroutineSM.NewObjectRef != s.execution.Object {
			// TODO: do nothing for now, later we should replace that mechanism
			panic(throw.NotImplemented())
		}

		s.lmnLastLifelineRef = reference.NewRecordOf(s.execution.Object, subroutineSM.NewLastLifelineRef.GetLocal())
		s.lmnIncomingRequestRef = subroutineSM.IncomingRequestRef

		return ctx.Jump(s.stepExecuteStart)
	})
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
		return ctx.Jump(s.stepOutboundRecord)
	case execution.Error, execution.Abort:
		ctx.Log().Error("execution failed", newState.Error)
		panic(throw.NotImplemented())
	default:
		panic(throw.IllegalValue())
	}

	return ctx.Jump(s.stepInboundResponseRecord)
}

func (s *SMVObjectTranscriptReport) stepOutboundRecord(ctx smachine.ExecutionContext) smachine.StateUpdate {
	switch outgoing := s.executionNewState.Outgoing.(type) {
	case execution.Deactivate:
		if s.execution.Isolation.Interference == isolation.CallIntolerable {
			panic(throw.NotImplemented())
		}

		if s.executionNewState.Result.SideEffectType != requestresult.SideEffectDeactivate {
			panic(throw.NotImplemented())
		}

		oldRequestResult := s.executionNewState.Result

		// we should overwrite old side effect with new one - deactivation of object
		s.executionNewState.Result = requestresult.New(oldRequestResult.Result(), oldRequestResult.ObjectReference())
		s.executionNewState.Result.SetDeactivate(s.execution.ObjectDescriptor)
		s.executionNewState.Type = execution.Done

		return ctx.Jump(s.stepInboundResponseRecord)
	case execution.CallConstructor:
		s.outgoingRequest = outgoing.ConstructVCallRequest(s.execution)
	case execution.CallMethod:
		s.outgoingRequest = outgoing.ConstructVCallRequest(s.execution)
	default:
		panic(throw.IllegalValue())
	}

	subroutineSM := s.constructSubSMRegister(RegisterOutgoingRequest)
	subroutineSM.Outgoing = s.outgoingRequest

	return ctx.CallSubroutine(&subroutineSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
		s.lmnLastLifelineRef = subroutineSM.NewLastLifelineRef
		s.lmnLastFilamentRef = subroutineSM.NewLastFilamentRef
		s.lmnIncomingRequestRef = subroutineSM.IncomingRequestRef

		s.outgoingRequest.CallOutgoing = rms.NewReference(s.lmnLastFilamentRef)

		return ctx.Jump(s.stepValidateIncomingRequest)
	})
}

func (s *SMVObjectTranscriptReport) stepValidateOutgoingRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
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
		ctx.Log().Warn("validation failed: wrong CallOutgoing")
		return ctx.Jump(s.stepValidationFailed)
	}

	return ctx.Jump(s.stepValidateOutgoingResult)
}

func (s *SMVObjectTranscriptReport) stepValidateOutgoingResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
	entry := s.findNextEntry()
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
	s.outgoingResultRef = outgoingResult.OutgoingResult.GetValue()

	subroutineSM := s.constructSubSMRegister(RegisterOutgoingResult)
	subroutineSM.OutgoingResult = s.outgoingResult

	return ctx.CallSubroutine(&subroutineSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
		s.lmnLastFilamentRef = subroutineSM.NewLastFilamentRef

		return ctx.Jump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
			equal := s.lmnLastFilamentRef.Equal(s.outgoingResultRef)
			if !equal {
				ctx.Log().Warn("validation failed: wrong OutgoingResult record reference")
				return ctx.Jump(s.stepValidationFailed)
			}

			return ctx.Jump(s.stepExecuteContinue)
		})
	})
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

func (s *SMVObjectTranscriptReport) stepInboundResponseRecord(ctx smachine.ExecutionContext) smachine.StateUpdate {
	subroutineSM := s.constructSubSMRegister(RegisterIncomingResult)
	subroutineSM.IncomingResult = s.executionNewState

	return ctx.CallSubroutine(&subroutineSM, nil, func(ctx smachine.SubroutineExitContext) smachine.StateUpdate {
		s.lmnLastLifelineRef = subroutineSM.NewLastLifelineRef
		s.lmnLastFilamentRef = subroutineSM.NewLastFilamentRef
		s.lmnIncomingRequestRef = subroutineSM.IncomingRequestRef

		if s.incomingValidated {
			return ctx.Jump(s.stepValidateIncomingResult)
		}
		return ctx.Jump(s.stepValidateIncomingRequest)
	})
}

func (s *SMVObjectTranscriptReport) stepValidateIncomingRequest(ctx smachine.ExecutionContext) smachine.StateUpdate {
	equalStartRecord := s.lmnIncomingRequestRef.Equal(s.incomingRecordRef.GetValue())
	if !equalStartRecord {
		ctx.Log().Warn("validation failed: wrong IncomingRequest record ref")
		return ctx.Jump(s.stepValidationFailed)
	}

	s.incomingValidated = true

	if s.outgoingRequest != nil {
		return ctx.Jump(s.stepValidateOutgoingRequest)
	}
	return ctx.Jump(s.stepValidateIncomingResult)
}

func (s *SMVObjectTranscriptReport) stepValidateIncomingResult(ctx smachine.ExecutionContext) smachine.StateUpdate {
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
	equal := s.lmnLastFilamentRef.Equal(callResult.IncomingResult.GetValue())
	if !equal {
		ctx.Log().Warn("validation failed: wrong IncomingResult record ref")
		return ctx.Jump(s.stepValidationFailed)
	}
	equalState := s.lmnLastLifelineRef.Equal(callResult.ObjectState.GetValue())
	if !equalState {
		ctx.Log().Warn("validation failed: wrong ObjectState record ref")
		return ctx.Jump(s.stepValidationFailed)
	}

	if s.validatedState.IsEmpty() {
		s.validatedState = s.lmnLastLifelineRef
	} else if s.incomingRequest.CallFlags.GetInterference() == isolation.CallTolerable {
		s.validatedState = s.lmnLastLifelineRef
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
	s.incomingValidated = false
	s.incomingRegistered = false

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

func (s *SMVObjectTranscriptReport) prepareExecution(ctx context.Context) {
	s.execution = execute.ExecContextFromRequest(s.incomingRequest)

	s.execution.Context = ctx
	s.execution.Pulse = s.pulseSlot.PulseData()
	s.execution.ObjectDescriptor = s.objDesc
	s.execution.Object = s.Payload.Object.GetValue()
}

const (
	RegisterLifeLine execute.RegisterVariant = iota
	RegisterOutgoingRequest
	RegisterOutgoingResult
	RegisterIncomingResult
)

func (s *SMVObjectTranscriptReport) constructSubSMRegister(v execute.RegisterVariant) lmn.SubSMRegister {
	subroutineSM := lmn.SubSMRegister{
		Interference: s.execution.Isolation.Interference,
		DryRun:       true,
		PulseNumber:  s.Payload.AsOf,
	}

	switch v {
	case RegisterLifeLine:
		if s.incomingRegistered {
			panic(throw.IllegalState())
		}
		subroutineSM.Incoming = s.incomingRequest
		return subroutineSM

	case RegisterOutgoingRequest, RegisterIncomingResult:
		if !s.incomingRegistered {
			subroutineSM.Incoming = s.incomingRequest

			s.incomingRegistered = true
		}

	case RegisterOutgoingResult:
		if !s.incomingRegistered {
			panic(throw.IllegalState())
		}

	default:
		panic(throw.IllegalValue())
	}

	subroutineSM.Object = s.execution.Object
	subroutineSM.LastLifelineRef = s.lmnLastLifelineRef
	subroutineSM.LastFilamentRef = s.lmnLastFilamentRef

	return subroutineSM
}
