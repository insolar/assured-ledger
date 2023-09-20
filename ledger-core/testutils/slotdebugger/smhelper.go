package slotdebugger

import (
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/testutils/predicate"
)

func NewStateMachineHelper(slotLink smachine.SlotLink) StateMachineHelper {
	return StateMachineHelper{SlotLinkFilter: predicate.NewSlotLinkFilter(slotLink)}
}

type StateMachineHelper struct {
	predicate.SlotLinkFilter
}

func (w StateMachineHelper) BeforeStep(fn smachine.StateFunc) predicate.Func {
	return predicate.And(w.IsFromSlot, predicate.BeforeStep(fn))
}

func (w StateMachineHelper) AfterStep(fn smachine.StateFunc) predicate.Func {
	return predicate.And(w.IsFromSlot, predicate.AfterStep(fn))
}

func (w StateMachineHelper) AfterAnyStep() predicate.Func {
	return predicate.And(w.IsFromSlot, predicate.AfterAnyStep())
}

func (w StateMachineHelper) AfterAnyMigrate() predicate.Func {
	return predicate.And(w.IsFromSlot, predicate.AfterAnyMigrate())
}

func (w StateMachineHelper) AfterMigrate(fn smachine.MigrateFunc) predicate.Func {
	return predicate.And(w.IsFromSlot, predicate.AfterMigrate(fn))
}

func (w StateMachineHelper) BeforeStepExt(s smachine.SlotStep) predicate.Func {
	return predicate.And(w.IsFromSlot, predicate.BeforeStepExt(s))
}

func (w StateMachineHelper) AfterStepExt(s smachine.SlotStep) predicate.Func {
	return predicate.And(w.IsFromSlot, predicate.AfterStepExt(s))
}

func (w StateMachineHelper) AfterTestString(marker string) predicate.Func {
	return predicate.And(w.IsFromSlot, predicate.AfterTestString(marker))
}

func (w StateMachineHelper) AfterCustomEvent(fn func(interface{}) bool) predicate.Func {
	return predicate.And(w.IsFromSlot, predicate.AfterCustomEvent(fn))
}

func (w StateMachineHelper) AfterCustomEventType(tp reflect.Type) predicate.Func {
	return predicate.And(w.IsFromSlot, predicate.AfterCustomEventType(tp))
}

func (w StateMachineHelper) AfterAsyncCall(id smachine.AdapterID) predicate.Func {
	return predicate.And(w.IsFromSlot, predicate.AfterAsyncCall(id))
}

func (w StateMachineHelper) AfterResultOfFirstAsyncCall(id smachine.AdapterID) predicate.Func {
	return predicate.And(w.IsFromSlot, predicate.AfterResultOfFirstAsyncCall(id))
}
