package smachine

import (
	"errors"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type subroutineMarker struct {
	slotID SlotID
	step   uint32
}

func (s *Slot) newSubroutineMarker() subroutineMarker {
	id, step, _ := s.GetState()
	return subroutineMarker{slotID: id, step: step}
}

func (s *Slot) getSubroutineMarker() subroutineMarker {
	if s.stateStack != nil {
		return s.stateStack.childMarker
	}
	return s.getTopSubroutineMarker()
}

func (s *Slot) getTopSubroutineMarker() subroutineMarker {
	id, _, _ := s.GetState()
	return subroutineMarker{slotID: id}
}

// func (s *Slot) ensureNoSubroutine() {
// 	if s.hasSubroutine() {
// 		panic(throw.IllegalState())
// 	}
// }

func (s *Slot) forceTopSubroutineUpdate(su StateUpdate, worker DetachableSlotWorker) StateUpdate {
	if !typeOfStateUpdate(su).IsSubroutineSafe() {
		s._popTillSubroutine(subroutineMarker{}, worker)
	}
	return su
}

func (s *Slot) forceSubroutineUpdate(su StateUpdate, producedBy subroutineMarker, worker DetachableSlotWorker) StateUpdate {
	switch isCurrent, isValid := s.checkSubroutineMarker(producedBy); {
	case isCurrent:
		//
	case !isValid:
		s.machine.logInternal(s.NewStepLink(), "aborting routine has expired", nil)
		return StateUpdate{}
	case !typeOfStateUpdate(su).IsSubroutineSafe():
		s._popTillSubroutine(producedBy, worker)
	}
	return su
}

var errTerminatedByCaller = errors.New("subroutine SM is terminated by caller")

func (s *Slot) _popTillSubroutine(producedBy subroutineMarker, worker DetachableSlotWorker) {
	for ms := s.stateStack; ms != nil; ms = ms.stateStack {
		if ms.childMarker == producedBy {
			return
		}
		s.applySubroutineStop(errTerminatedByCaller, worker)
	}
	if producedBy.step != 0 {
		panic(throw.IllegalState())
	}
}

func (s *Slot) hasSubroutine() bool {
	return s.stateStack != nil
}

func (s *Slot) checkSubroutineMarker(marker subroutineMarker) (isCurrent, isValid bool) {
	switch id := s.GetSlotID(); {
	case id != marker.slotID:
		//
	case marker.step == 0:
		return s.stateStack == nil, true
	case s.stateStack == nil:
		//
	case s.stateStack.childMarker == marker:
		return true, true
	default:
		for sbs := s.stateStack.stateStack; sbs != nil; sbs = sbs.stateStack {
			if sbs.childMarker == marker {
				return false, true
			}
		}
	}
	return false, false
}

func (s *Slot) prepareSubroutineStart(ssm SubroutineStateMachine, exitFn SubroutineExitFunc, migrateFn MigrateFunc) SlotStep {
	if exitFn == nil {
		panic(throw.IllegalValue())
	}
	if ssm == nil {
		panic(throw.IllegalValue())
	}
	decl := ssm.GetStateMachineDeclaration()
	if decl == nil {
		panic(throw.IllegalValue())
	}

	return SlotStep{Migration: migrateFn, Transition: func(ctx ExecutionContext) StateUpdate {
		ec := ctx.(*executionContext)
		slot := ec.s
		tracerID := ctx.Log().GetTracerID()

		prev := slot.stateMachineData
		if migrateFn == nil {
			migrateFn = prev.defMigrate
		}

		stackedSdd := &stackedStateMachineData{prev, migrateFn,
			exitFn, slot.newSubroutineMarker(),
			nil, 0,
			prev.stateStack != nil && prev.stateStack.hasMigrates,
		}
		if stackedSdd.stackMigrate != nil {
			stackedSdd.hasMigrates = true
		}

		slot.stateMachineData = stateMachineData{declaration: decl, stateStack: stackedSdd}
		initFn := slot.prepareSubroutineInit(ssm, tracerID)

		return initFn.defaultInit(ctx)
	}}
}

var defaultSubroutineStartDecl = StepDeclaration{stepDeclExt: stepDeclExt{Name: "<init_subroutine>"}}
var defaultSubroutineExitDecl = StepDeclaration{stepDeclExt: stepDeclExt{Name: "<exit_subroutine>"}}

func (s *Slot) applySubroutineStop(lastError error, worker DetachableSlotWorker) {
	s.step = s.prepareSubroutineStop(lastError, worker)
	s.stepDecl = &defaultSubroutineExitDecl
}

func (s *Slot) prepareSubroutineStop(lastError error, worker DetachableSlotWorker) SlotStep {
	prev := s.stateStack
	if prev == nil {
		panic(throw.IllegalState())
	}

	lastError = s.callFinalizeOnce(worker, lastError)

	lastResult := s.defResult
	returnFn := prev.returnFn

	s.stateMachineData = prev.stateMachineData
	s.restoreSubroutineAliases(prev.copyAliases, prev.cleanupMode)

	// TODO logging
	bc := subroutineExitContext{
		bargingInContext{ slotContext{s: s, w: worker}, false, false},
		lastResult, lastError}
	su := bc.executeSubroutineExit(returnFn)

	return SlotStep{Transition: func(ctx ExecutionContext) StateUpdate {
		ec := ctx.(*executionContext)
		su.marker = ec.getMarker()
		return su
	}}
}
