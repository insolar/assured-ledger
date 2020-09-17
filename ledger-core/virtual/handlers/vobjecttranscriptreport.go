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
	case execution.Error, execution.Abort, execution.OutgoingCall:
		panic(throw.NotImplemented())
	default:
		panic(throw.IllegalValue())
	}

	var newDesc descriptor.Object

	switch s.executionNewState.Result.Type() {
	case requestresult.SideEffectNone:
	case requestresult.SideEffectActivate:
		_, class, memory := s.executionNewState.Result.Activate()
		newDesc = s.makeNewDescriptor(class, memory, false)
	case requestresult.SideEffectAmend:
		_, class, memory := s.executionNewState.Result.Amend()
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
	if equal {
		s.validatedState = stateRef
		return ctx.Jump(s.stepAdvanceToNextRequest)
	}

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
	objDescriptor := s.execution.ObjectDescriptor
	if objDescriptor != nil {
		prevStateIDBytes = objDescriptor.StateID().AsBytes()
	}

	objectRefBytes := s.execution.Object.AsBytes()
	stateHash := append(memory, objectRefBytes...)
	stateHash = append(stateHash, prevStateIDBytes...)

	stateID := execute.NewStateID(s.pulseSlot.PulseData().GetPulseNumber(), stateHash)
	return descriptor.NewObject(
		s.execution.Object,
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
