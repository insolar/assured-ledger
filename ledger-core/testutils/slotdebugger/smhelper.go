// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package slotdebugger

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	testUtilsCommon "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/utils"
)

func NewStateMachineHelper(sm smachine.StateMachine, slotLink smachine.SlotLink) StateMachineHelper {
	return StateMachineHelper{sm: sm, slotLink: slotLink}
}

type StateMachineHelper struct {
	sm       smachine.StateMachine
	slotLink smachine.SlotLink
}

func (w StateMachineHelper) IsValid() bool {
	return w.slotLink.IsValid()
}

func (w StateMachineHelper) StateMachine() smachine.StateMachine {
	return w.sm
}

func (w StateMachineHelper) SlotLink() smachine.SlotLink {
	return w.slotLink
}

func (w StateMachineHelper) BeforeStep(fn smachine.StateFunc) func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		switch {
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
		case event.Data.EventType != smachine.StepLoggerUpdate && event.Data.EventType != smachine.StepLoggerMigrate:
		case event.Update.NextStep.Transition == nil:
			// Transition == nil means that the step remains the same
			return utils.CmpStateFuncs(fn, event.Data.CurrentStep.Transition)
		default:
			return utils.CmpStateFuncs(fn, event.Update.NextStep.Transition)
		}
		return false
	}
}

func (w StateMachineHelper) AfterStep(fn smachine.StateFunc) func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		if event.Data.EventType != smachine.StepLoggerUpdate || w.slotLink.SlotID() != event.Data.StepNo.SlotID() {
			return false
		}
		return utils.CmpStateFuncs(fn, event.Data.CurrentStep.Transition)
	}
}

func (w StateMachineHelper) AfterAnyStep() func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		return event.Data.EventType == smachine.StepLoggerUpdate && w.slotLink.SlotID() == event.Data.StepNo.SlotID()
	}
}

func (w StateMachineHelper) AfterAnyMigrate() func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		switch {
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
		case event.Data.EventType != smachine.StepLoggerMigrate:
		default:
			return true
		}
		return false
	}
}

func (w StateMachineHelper) AfterStop() func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		return !w.slotLink.IsValid()
	}
}

func (w StateMachineHelper) AfterMigrate(fn smachine.MigrateFunc) func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		if event.Data.EventType != smachine.StepLoggerMigrate || w.slotLink.SlotID() != event.Data.StepNo.SlotID() {
			return false
		}
		return utils.CmpStateFuncs(fn, event.Update.AppliedMigrate)
	}
}

func (w StateMachineHelper) BeforeStepExt(s smachine.SlotStep) func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		step := event.Update.NextStep
		switch {
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
		case event.Data.EventType != smachine.StepLoggerUpdate && event.Data.EventType != smachine.StepLoggerMigrate:
		case s.Transition != nil && !utils.CmpStateFuncs(s.Transition, step.Transition):
		case s.Migration != nil && !utils.CmpStateFuncs(s.Migration, step.Migration):
		case s.Handler != nil && !utils.CmpStateFuncs(s.Handler, step.Handler):
		case step.Flags&s.Flags != s.Flags:
		case s.Transition == nil:
			return true
		case step.Transition == nil:
			// Transition == nil means that the step remains the same
			return utils.CmpStateFuncs(s.Transition, event.Data.CurrentStep.Transition)
		default:
			return utils.CmpStateFuncs(s.Transition, step.Transition)
		}
		return false
	}
}

func (w StateMachineHelper) AfterStepExt(s smachine.SlotStep) func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		step := event.Data.CurrentStep
		switch {
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
		case event.Data.EventType != smachine.StepLoggerUpdate:
		case s.Transition != nil && !utils.CmpStateFuncs(s.Transition, step.Transition):
		case s.Migration != nil && !utils.CmpStateFuncs(s.Migration, step.Migration):
		case s.Handler != nil && !utils.CmpStateFuncs(s.Handler, step.Handler):
		case step.Flags&s.Flags != s.Flags:
		case s.Transition == nil:
			return true
		default:
			return utils.CmpStateFuncs(s.Transition, step.Transition)
		}
		return false
	}
}

func (w StateMachineHelper) AfterCustomEvent(fn func(interface{}) bool) func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		switch {
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
		case !event.Data.EventType.IsEvent():
		default:
			return fn(event.CustomEvent)
		}
		return false
	}
}

func (w StateMachineHelper) AfterAsyncCall(id smachine.AdapterID) func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		switch {
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
		case event.Data.EventType != smachine.StepLoggerAdapterCall:
		case event.AdapterID != id:
		case event.Data.Flags.AdapterFlags() != smachine.StepLoggerAdapterAsyncCall:
		default:
			return true
		}
		return false
	}
}

func (w StateMachineHelper) AfterResultOfFirstAsyncCall(id smachine.AdapterID) func(testUtilsCommon.UpdateEvent) bool {
	hasCall := false
	callID := uint64(0)
	return func(event testUtilsCommon.UpdateEvent) bool {
		switch {
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
		case event.Data.EventType != smachine.StepLoggerAdapterCall:
		case event.AdapterID != id:
		case !hasCall:
			if event.Data.Flags.AdapterFlags() == smachine.StepLoggerAdapterAsyncCall {
				hasCall = true
				callID = event.CallID
			}
		case callID != event.CallID:
		default:
			switch event.Data.Flags.AdapterFlags() {
			case smachine.StepLoggerAdapterAsyncCall:
				panic(throw.FailHere("duplicate async call id"))
			case smachine.StepLoggerAdapterAsyncCancel:
				panic(throw.FailHere("async call was cancelled"))
			case smachine.StepLoggerAdapterAsyncResult:
				return true
			}
		}
		return false
	}
}
