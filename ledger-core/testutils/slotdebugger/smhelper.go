// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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
	return predicate.And(w.IsSlotLink, predicate.BeforeStep(fn))
}

func (w StateMachineHelper) AfterStep(fn smachine.StateFunc) predicate.Func {
	return predicate.And(w.IsSlotLink, predicate.AfterStep(fn))
}

func (w StateMachineHelper) AfterAnyStep() predicate.Func {
	return predicate.And(w.IsSlotLink, predicate.AfterAnyStep())
}

func (w StateMachineHelper) AfterAnyMigrate() predicate.Func {
	return predicate.And(w.IsSlotLink, predicate.AfterAnyMigrate())
}

func (w StateMachineHelper) AfterMigrate(fn smachine.MigrateFunc) predicate.Func {
	return predicate.And(w.IsSlotLink, predicate.AfterMigrate(fn))
}

func (w StateMachineHelper) BeforeStepExt(s smachine.SlotStep) predicate.Func {
	return predicate.And(w.IsSlotLink, predicate.BeforeStepExt(s))
}

func (w StateMachineHelper) AfterStepExt(s smachine.SlotStep) predicate.Func {
	return predicate.And(w.IsSlotLink, predicate.AfterStepExt(s))
}

func (w StateMachineHelper) AfterTestString(marker string) predicate.Func {
	return predicate.And(w.IsSlotLink, predicate.AfterTestString(marker))
}

func (w StateMachineHelper) AfterCustomEvent(fn func(interface{}) bool) predicate.Func {
	return predicate.And(w.IsSlotLink, predicate.AfterCustomEvent(fn))
}

func (w StateMachineHelper) AfterCustomEventType(tp reflect.Type) predicate.Func {
	return predicate.And(w.IsSlotLink, predicate.AfterCustomEventType(tp))
}

func (w StateMachineHelper) AfterAsyncCall(id smachine.AdapterID) predicate.Func {
	return predicate.And(w.IsSlotLink, predicate.AfterAsyncCall(id))
}

func (w StateMachineHelper) AfterResultOfFirstAsyncCall(id smachine.AdapterID) predicate.Func {
	return predicate.And(w.IsSlotLink, predicate.AfterResultOfFirstAsyncCall(id))
}
