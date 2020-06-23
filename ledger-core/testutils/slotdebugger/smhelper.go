// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package slotdebugger

import (
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/debuglogger"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
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

func (w StateMachineHelper) BeforeStep(fn smachine.StateFunc) func(debuglogger.UpdateEvent) bool {
	return func(event debuglogger.UpdateEvent) bool {
		switch {
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
		case event.Data.EventType != smachine.StepLoggerUpdate && event.Data.EventType != smachine.StepLoggerMigrate:
		case event.Update.NextStep.Transition == nil:
			// Transition == nil means that the step remains the same
			return testutils.CmpStateFuncs(fn, event.Data.CurrentStep.Transition)
		default:
			return testutils.CmpStateFuncs(fn, event.Update.NextStep.Transition)
		}
		return false
	}
}

func (w StateMachineHelper) AfterStep(fn smachine.StateFunc) func(debuglogger.UpdateEvent) bool {
	return func(event debuglogger.UpdateEvent) bool {
		if event.Data.EventType != smachine.StepLoggerUpdate || w.slotLink.SlotID() != event.Data.StepNo.SlotID() {
			return false
		}
		return testutils.CmpStateFuncs(fn, event.Data.CurrentStep.Transition)
	}
}

func (w StateMachineHelper) AfterAnyStep() func(debuglogger.UpdateEvent) bool {
	return func(event debuglogger.UpdateEvent) bool {
		return event.Data.EventType == smachine.StepLoggerUpdate && w.slotLink.SlotID() == event.Data.StepNo.SlotID()
	}
}

func (w StateMachineHelper) AfterAnyMigrate() func(debuglogger.UpdateEvent) bool {
	return func(event debuglogger.UpdateEvent) bool {
		switch {
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
		case event.Data.EventType != smachine.StepLoggerMigrate:
		default:
			return true
		}
		return false
	}
}

func (w StateMachineHelper) AfterStop() func(debuglogger.UpdateEvent) bool {
	return func(event debuglogger.UpdateEvent) bool {
		return !w.slotLink.IsValid()
	}
}

func (w StateMachineHelper) AfterMigrate(fn smachine.MigrateFunc) func(debuglogger.UpdateEvent) bool {
	return func(event debuglogger.UpdateEvent) bool {
		if event.Data.EventType != smachine.StepLoggerMigrate || w.slotLink.SlotID() != event.Data.StepNo.SlotID() {
			return false
		}
		return testutils.CmpStateFuncs(fn, event.Update.AppliedMigrate)
	}
}

func (w StateMachineHelper) BeforeStepExt(s smachine.SlotStep) func(debuglogger.UpdateEvent) bool {
	return func(event debuglogger.UpdateEvent) bool {
		step := event.Update.NextStep
		switch {
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
		case event.Data.EventType != smachine.StepLoggerUpdate && event.Data.EventType != smachine.StepLoggerMigrate:
		case s.Transition != nil && !testutils.CmpStateFuncs(s.Transition, step.Transition):
		case s.Migration != nil && !testutils.CmpStateFuncs(s.Migration, step.Migration):
		case s.Handler != nil && !testutils.CmpStateFuncs(s.Handler, step.Handler):
		case step.Flags&s.Flags != s.Flags:
		case s.Transition == nil:
			return true
		case step.Transition == nil:
			// Transition == nil means that the step remains the same
			return testutils.CmpStateFuncs(s.Transition, event.Data.CurrentStep.Transition)
		default:
			return testutils.CmpStateFuncs(s.Transition, step.Transition)
		}
		return false
	}
}

func (w StateMachineHelper) AfterStepExt(s smachine.SlotStep) func(debuglogger.UpdateEvent) bool {
	return func(event debuglogger.UpdateEvent) bool {
		step := event.Data.CurrentStep
		switch {
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
		case event.Data.EventType != smachine.StepLoggerUpdate:
		case s.Transition != nil && !testutils.CmpStateFuncs(s.Transition, step.Transition):
		case s.Migration != nil && !testutils.CmpStateFuncs(s.Migration, step.Migration):
		case s.Handler != nil && !testutils.CmpStateFuncs(s.Handler, step.Handler):
		case step.Flags&s.Flags != s.Flags:
		case s.Transition == nil:
			return true
		default:
			return testutils.CmpStateFuncs(s.Transition, step.Transition)
		}
		return false
	}
}

func (w StateMachineHelper) AfterTestString(marker string) func(debuglogger.UpdateEvent) bool {
	return func(event debuglogger.UpdateEvent) bool {
		switch {
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
		case event.Data.EventType != smachine.StepLoggerTrace:
		default:
			if s, ok := event.CustomEvent.(string); ok {
				return s == marker
			}
		}
		return false
	}
}

func (w StateMachineHelper) AfterCustomEvent(fn func(interface{}) bool) func(debuglogger.UpdateEvent) bool {
	return func(event debuglogger.UpdateEvent) bool {
		switch {
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
		case !event.Data.EventType.IsEvent():
		default:
			return fn(event.CustomEvent)
		}
		return false
	}
}

func (w StateMachineHelper) AfterCustomEventType(tp reflect.Type) func(debuglogger.UpdateEvent) bool {
	return func(event debuglogger.UpdateEvent) bool {
		switch {
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
		case !event.Data.EventType.IsEvent():
		default:
			return reflect.ValueOf(event.CustomEvent).Type().AssignableTo(tp)
		}
		return false
	}
}

func (w StateMachineHelper) AfterAsyncCall(id smachine.AdapterID) func(debuglogger.UpdateEvent) bool {
	return func(event debuglogger.UpdateEvent) bool {
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

func (w StateMachineHelper) AfterResultOfFirstAsyncCall(id smachine.AdapterID) func(debuglogger.UpdateEvent) bool {
	hasCall := false
	callID := uint64(0)
	return func(event debuglogger.UpdateEvent) bool {
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
			case smachine.StepLoggerAdapterAsyncExpiredCancel, smachine.StepLoggerAdapterAsyncExpiredResult:
				panic(throw.FailHere("async callback has expired"))
			case smachine.StepLoggerAdapterAsyncResult:
				return true
			}
		}
		return false
	}
}
