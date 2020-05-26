// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package slotdebugger

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	testUtilsCommon "github.com/insolar/assured-ledger/ledger-core/v2/testutils"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils/utils"
)

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

func (w StateMachineHelper) TilStep(fn smachine.StateFunc) func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		if event.Data.EventType != smachine.StepLoggerUpdate || w.slotLink.SlotID() != event.Data.StepNo.SlotID() {
			return false
		}
		return utils.CmpStateFuncs(fn, event.Update.NextStep.Transition)
	}
}

func (w StateMachineHelper) TilAnyStep() func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		return event.Data.EventType == smachine.StepLoggerUpdate && w.slotLink.SlotID() == event.Data.StepNo.SlotID()
	}
}

func (w StateMachineHelper) TilAnyMigrate() func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		return event.Data.EventType == smachine.StepLoggerMigrate && w.slotLink.SlotID() == event.Data.StepNo.SlotID()
	}
}

func (w StateMachineHelper) TilStop() func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		return !w.slotLink.IsValid()
	}
}

func (w StateMachineHelper) TilMigrate(fn smachine.MigrateFunc) func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		if event.Data.EventType != smachine.StepLoggerMigrate || w.slotLink.SlotID() != event.Data.StepNo.SlotID() {
			return false
		}
		return utils.CmpStateFuncs(fn, event.Update.AppliedMigrate)
	}
}

func (w StateMachineHelper) TilStepExt(s smachine.SlotStep) func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		switch {
		case event.Data.EventType != smachine.StepLoggerUpdate:
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
		case s.Transition != nil && !utils.CmpStateFuncs(s.Transition, event.Update.NextStep.Transition):
		case s.Migration != nil && !utils.CmpStateFuncs(s.Migration, event.Update.NextStep.Migration):
		case s.Handler != nil && !utils.CmpStateFuncs(s.Handler, event.Update.NextStep.Handler):
		default:
			return event.Update.NextStep.Flags&s.Flags == s.Flags
		}
		return false
	}
}

func (w StateMachineHelper) TilCustomEvent(fn func(interface{}) bool) func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		switch {
		case !event.Data.EventType.IsEvent():
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
		default:
			return fn(event.CustomEvent)
		}
		return false
	}
}

func (w StateMachineHelper) TilAsyncCall(id smachine.AdapterID) func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		switch {
		case event.Data.EventType != smachine.StepLoggerAdapterCall:
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
		case event.AdapterID != id:
		case event.Data.Flags.AdapterFlags() != smachine.StepLoggerAdapterAsyncCall:
		default:
			return true
		}
		return false
	}
}

func (w StateMachineHelper) TilResultOfFirstAsyncCall(id smachine.AdapterID) func(testUtilsCommon.UpdateEvent) bool {
	hasCall := false
	callID := uint64(0)
	return func(event testUtilsCommon.UpdateEvent) bool {
		switch {
		case event.Data.EventType != smachine.StepLoggerAdapterCall:
		case w.slotLink.SlotID() != event.Data.StepNo.SlotID():
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

