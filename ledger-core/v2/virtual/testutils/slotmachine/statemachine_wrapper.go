// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package slotmachine

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	testUtilsCommon "github.com/insolar/assured-ledger/ledger-core/v2/testutils"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/testutils/utils"
)

type StateMachineWrapper struct {
	sm       smachine.StateMachine
	slotLink smachine.SlotLink
}

func NewStateMachineWrapper(sm smachine.StateMachine, slotLink smachine.SlotLink) *StateMachineWrapper {
	return &StateMachineWrapper{
		sm:       sm,
		slotLink: slotLink,
	}
}

func (w *StateMachineWrapper) StateMachine() smachine.StateMachine {
	return w.sm
}

func (w *StateMachineWrapper) SlotLink() smachine.SlotLink {
	return w.slotLink
}

func (w *StateMachineWrapper) WaitStep(fn smachine.StateFunc) func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		if w.slotLink.SlotID() != event.Data.StepNo.SlotID() {
			return false
		}
		return utils.CmpStateFuncs(fn, event.Update.NextStep.Transition)
	}
}

func (w *StateMachineWrapper) WaitMigrate(fn smachine.MigrateFunc) func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		if w.slotLink.SlotID() != event.Data.StepNo.SlotID() {
			return false
		}
		return utils.CmpStateFuncs(fn, event.Update.AppliedMigrate)
	}
}

func (w *StateMachineWrapper) WaitStepExt(s smachine.SlotStep) func(testUtilsCommon.UpdateEvent) bool {
	return func(event testUtilsCommon.UpdateEvent) bool {
		if w.slotLink.SlotID() != event.Data.StepNo.SlotID() {
			return false
		}
		if s.Transition != nil {
			panic(throw.FailHere("Transition is empty"))
		}
		if !utils.CmpStateFuncs(s.Transition, event.Update.NextStep.Transition) {
			return false
		}
		if s.Migration != nil && !utils.CmpStateFuncs(s.Transition, event.Update.NextStep.Migration) {
			return false
		}
		if s.Handler != nil && !utils.CmpStateFuncs(s.Transition, event.Update.NextStep.Handler) {
			return false
		}
		return event.Update.NextStep.Flags&s.Flags == s.Flags
	}
}
