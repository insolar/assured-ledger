// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"errors"
	"runtime"
)

const (
	stateUpdNoChange stateUpdKind = iota // must be zero to match StateUpdate.IsEmpty()

	stateUpdStop
	stateUpdError
	stateUpdPanic
	stateUpdReplace
	stateUpdSubroutineBegin
	stateUpdSubroutineEnd

	stateUpdInternalRepeatNow // this is a special op. MUST NOT be used anywhere else.

	stateUpdRepeat   // supports short-loop
	stateUpdNextLoop // supports short-loop

	stateUpdWakeup
	stateUpdNext
	stateUpdRestore
	stateUpdPoll
	stateUpdSleep
	stateUpdWaitForEvent
	stateUpdWaitForActive
	stateUpdWaitForIdle
)

var stateUpdateTypes []StateUpdateType

// init() is used instead of variable initializer to avoid "initialization loop" error
func init() {
	stateUpdateTypes = []StateUpdateType{
		stateUpdNoChange: {
			name:   "noop",
			filter: updCtxMigrate | updCtxBargeIn | updCtxAsyncCallback,

			safeWithSubroutine: true,

			apply: func(slot *Slot, stateUpdate StateUpdate, worker FixedSlotWorker, _ *StepDeclaration) (isAvailable bool, err error) {
				return true, nil
			},
		},

		stateUpdInternalRepeatNow: {
			name:   "repeatNow",
			filter: updCtxInternal, // can't be created by a template

			apply: func(slot *Slot, stateUpdate StateUpdate, worker FixedSlotWorker, _ *StepDeclaration) (isAvailable bool, err error) {
				if slot.isInQueue() {
					return false, errors.New("unexpected internal repeat")
				}
				m := slot.machine
				m.workingSlots.AddFirst(slot)
				return true, nil
			},
		},

		stateUpdStop: {
			name:   "stop",
			filter: updCtxExec | updCtxInit | updCtxMigrate | updCtxBargeIn | updCtxSubrExit,

			// prepare: func(slot *Slot, stateUpdate *StateUpdate) {
			// 	if slot.hasSubroutine() {
			// 		stateUpdate.step = slot.prepareSubroutineStop(nil, worker)
			// 		stateUpdate.updKind = uint8(stateUpdSubroutineEnd)
			// 		return
			// 	}
			//
			// 	if err := slot.callFinalizeOnce(worker, nil); err != nil {
			// 		stateUpdate.param1 = err
			// 		stateUpdate.updKind = uint8(stateUpdPanic)
			// 	}
			// },
			//
			// apply: func(slot *Slot, _ StateUpdate, worker FixedSlotWorker, _ *StepDeclaration) (isAvailable bool, err error) {
			// 	slot.machine.recycleSlot(slot, worker)
			// 	return false, nil
			// },

			apply:  stateUpdateDefaultStop,
		},

		stateUpdError: {
			name:      "error",
			filter:    updCtxExec | updCtxInit | updCtxMigrate | updCtxBargeIn | updCtxSubrExit,
			params:    updParamVar,
			varVerify: stateUpdateDefaultVerifyError,
			apply: func(slot *Slot, stateUpdate StateUpdate, worker FixedSlotWorker, _ *StepDeclaration) (isAvailable bool, err error) {
				err = stateUpdate.param1.(error)
				if err == nil {
					err = errors.New("error argument is missing")
				}
				return slot.machine.handleSlotUpdateError(slot, worker, stateUpdate, false, err), nil
			},
		},

		stateUpdPanic: {
			name:      "panic",
			filter:    updCtxInternal, // can't be created by a template
			params:    updParamVar,
			varVerify: stateUpdateDefaultVerifyError,
			apply: func(slot *Slot, stateUpdate StateUpdate, worker FixedSlotWorker, _ *StepDeclaration) (isAvailable bool, err error) {
				err = stateUpdate.param1.(error)
				if err == nil {
					err = errors.New("error argument is missing")
				}
				return slot.machine.handleSlotUpdateError(slot, worker, stateUpdate, true, err), nil
			},
		},

		stateUpdReplace: {
			name:   "replace",
			filter: updCtxExec,
			params: updParamStep,

			stepDeclaration: &replaceInitDecl,

			prepare: func(slot *Slot, _ *StateUpdate) {
				slot.slotFlags |= slotStepSuspendMigrate
			},

			apply: stateUpdateDefaultJump,
		},

		stateUpdSubroutineBegin: {
			name:   "subroutineBegin",
			filter: updCtxExec|updCtxMigrate,
			params: updParamStep,

			stepDeclaration: &defaultSubroutineStartDecl,

			apply: stateUpdateDefaultJump,
		},

		stateUpdSubroutineEnd: {
			name:   "subroutineEnd",
			filter: updCtxInternal,
			params: updParamStep,

			stepDeclaration: &defaultSubroutineExitDecl,

			apply: stateUpdateDefaultJump,
		},

		stateUpdRepeat: {
			name:   "repeat",
			filter: updCtxExec,
			params: updParamUint,

			safeWithSubroutine: true,

			shortLoop: func(slot *Slot, stateUpdate StateUpdate, loopCount uint32, _ *StepDeclaration) bool {
				return loopCount < stateUpdate.param0
			},

			apply: func(slot *Slot, stateUpdate StateUpdate, worker FixedSlotWorker, _ *StepDeclaration) (isAvailable bool, err error) {
				slot.activateSlot(worker)
				return true, nil
			},
		},

		stateUpdNextLoop: {
			name:   "jumpLoop",
			filter: updCtxExec,
			params: updParamStep | updParamUint,

			shortLoop: func(slot *Slot, stateUpdate StateUpdate, loopCount uint32, nextDecl *StepDeclaration) bool {
				if loopCount >= stateUpdate.param0 {
					return false
				}

				nextStep := stateUpdate.step.Transition
				if nextStep == nil {
					return false // the same step won't be short-looped
				}

				if nextDecl == nil {
					nextDecl = slot.declaration.GetStepDeclaration(nextStep)
				}

				curStep := slot.step.Transition
				prevSeqID := 0
				if slot.stepDecl != nil {
					prevSeqID = slot.stepDecl.SeqID
				}
				slot.setNextStep(stateUpdate.step, nextDecl)

				if nextDecl != nil && prevSeqID != 0 {
					if nextSeqID := nextDecl.SeqID; nextSeqID != 0 {
						return nextSeqID > prevSeqID // only proper forward steps can be short-looped
					}
				}
				isConsecutive := slot.declaration.IsConsecutive(curStep, nextStep)
				return isConsecutive
			},

			apply: stateUpdateDefaultJump,
		},

		stateUpdWakeup: {
			name:   "wakeUp",
			filter: updCtxExec | updCtxBargeIn | updCtxAsyncCallback | updCtxMigrate,

			safeWithSubroutine: true,

			apply: func(slot *Slot, stateUpdate StateUpdate, worker FixedSlotWorker, _ *StepDeclaration) (isAvailable bool, err error) {
				slot.activateSlot(worker)
				return true, nil
			},
		},

		stateUpdNext: {
			name:      "jump",
			filter:    updCtxExec | updCtxInit | updCtxBargeIn | updCtxMigrate | updCtxSubrExit,
			params:    updParamStep | updParamVar,
			prepare:   stateUpdateDefaultNoArgPrepare,
			varVerify: stateUpdateDefaultVerifyNoArgFn,
			apply:     stateUpdateDefaultJump,
		},

		stateUpdRestore: {
			name:      "restore",
			filter:    updCtxExec | updCtxInit | updCtxBargeIn | updCtxMigrate | updCtxSubrExit,
			params:    updParamStep,
			apply: func(slot *Slot, stateUpdate StateUpdate, worker FixedSlotWorker, sd *StepDeclaration) (isAvailable bool, err error) {
				m := slot.machine
				slot.setNextStep(stateUpdate.step, sd)
				if stateUpdate.step.Flags & stepSleepState == 0 {
					m.updateSlotQueue(slot, worker, activateSlot)
				}
				return true, nil
			},
		},

		stateUpdPoll: {
			name:    "poll",
			filter:  updCtxExec,
			params:  updParamStep | updParamVar,
			prepare: stateUpdateDefaultNoArgPrepare,
			apply: func(slot *Slot, stateUpdate StateUpdate, worker FixedSlotWorker, _ *StepDeclaration) (isAvailable bool, err error) {
				m := slot.machine
				slot.setNextStep(stateUpdate.step, nil)
				m.updateSlotQueue(slot, worker, deactivateSlot)
				m.pollingSlots.AddToLatest(slot)
				return true, nil
			},
		},

		stateUpdSleep: {
			name:    "sleep",
			filter:  updCtxExec,
			params:  updParamStep | updParamVar,
			prepare: stateUpdateDefaultNoArgPrepare,
			apply: func(slot *Slot, stateUpdate StateUpdate, worker FixedSlotWorker, _ *StepDeclaration) (isAvailable bool, err error) {
				m := slot.machine
				slot.setNextStep(stateUpdate.step, nil)
				m.updateSlotQueue(slot, worker, deactivateSlot)
				return true, nil
			},
		},

		stateUpdWaitForEvent: {
			name:    "waitEvent",
			filter:  updCtxExec,
			params:  updParamStep | updParamUint | updParamVar,
			prepare: stateUpdateDefaultNoArgPrepare,
			apply: func(slot *Slot, stateUpdate StateUpdate, worker FixedSlotWorker, _ *StepDeclaration) (isAvailable bool, err error) {
				m := slot.machine
				slot.setNextStep(stateUpdate.step, nil)

				if stateUpdate.param0 == 0 {
					m.updateSlotQueue(slot, worker, activateHotWaitSlot)
					return true, nil
				}

				waitUntil := m.fromRelativeTime(stateUpdate.param0)

				if !m.scanStartedAt.Before(waitUntil) { // not(started<Until) ==> started>=Until
					slot.activateSlot(worker)
					return true, nil
				}

				m.updateSlotQueue(slot, worker, deactivateSlot)
				if !m.pollingSlots.AddToLatestBefore(waitUntil, slot) {
					m.scanWakeUpAt = minTime(m.scanWakeUpAt, waitUntil)
					m.updateSlotQueue(slot, worker, activateHotWaitSlot)
				}

				return true, nil
			},
		},

		stateUpdWaitForActive: {
			name:   "waitActive",
			filter: updCtxExec,
			params: updParamStep | updParamLink,
			//		prepare: stateUpdateDefaultNoArgPrepare,
			apply: func(slot *Slot, stateUpdate StateUpdate, worker FixedSlotWorker, _ *StepDeclaration) (isAvailable bool, err error) {
				m := slot.machine
				slot.setNextStep(stateUpdate.step, nil)
				waitOn := stateUpdate.getLink()

				if waitOn.s == slot {
					// don't wait for self
					slot.activateSlot(worker)
					return true, nil
				}

				m.updateSlotQueue(slot, worker, deactivateSlot)

				// TODO work in progress
				panic("work in progress")

				// switch isValid, isBusy := waitOn.getIsValidAndBusy(); {
				// case !isValid:
				//	// don't wait for an expired or busy slot
				//	m.updateSlotQueue(slot, worker, activateSlot)
				//	return true, nil
				// case waitOn.isMachine(m):
				//	if isBusy {
				//		m.updateSlotQueue(slot, worker, activateHotWaitSlot)
				//		return true, nil
				//	}
				//	//m.queueOnSlot(waitOn.s, slot)
				//	panic("not implemented")
				// case worker.OuterCall(waitOn.s.machine, func(worker FixedSlotWorker) {
				//
				// }):
				// default:
				//	panic("not implemented") // TODO decide on action
				// }

				// switch waitOn.s.QueueType() {
				// case ActiveSlots, WorkingSlots:
				//	// don't wait
				//	m.updateSlotQueue(slot, worker, activateSlot)
				// case NoQueue:
				//	waitOn.s.makeQueueHead()
				//	fallthrough
				// case ActivationOfSlot, PollingSlots:
				//	m.updateSlotQueue(slot, worker, deactivateSlot)
				//	waitOn.s.queue.AddLast(slot)
				// default:
				//	return false, errors.New("illegal slot queue")
				// }
				// return true, nil
			},
		},

		stateUpdWaitForIdle: {
			name:    "waitIdle",
			filter:  updCtxExec,
			params:  updParamStep | updParamLink,
			prepare: stateUpdateDefaultNoArgPrepare,
			apply: func(slot *Slot, stateUpdate StateUpdate, worker FixedSlotWorker, _ *StepDeclaration) (isAvailable bool, err error) {
				m := slot.machine
				slot.setNextStep(stateUpdate.step, nil)

				waitOn := stateUpdate.getLink()
				mWaitOn := waitOn.getActiveMachine()
				if waitOn.s == slot || !waitOn.IsValid() || mWaitOn == nil {
					// don't wait for self
					// don't wait for an expired slot
					slot.activateSlot(worker)
					return true, nil
				}

				m.updateSlotQueue(slot, worker, deactivateSlot)

				wakeupLink := slot.NewLink()
				// here is a trick - we put a callback on the AWAITED object
				// because a callback is executed on non-busy object only
				// hence our call back will only be triggered when the object became available
				if !mWaitOn.syncQueue.AddAsyncCallback(waitOn, wakeupLink.activateOnNonBusy) {
					// callback was declined - wake up the requested back
					slot.activateSlot(worker)
				}
				return true, nil
			},
		},
	}

	for i := range stateUpdateTypes {
		if stateUpdateTypes[i].filter != 0 {
			stateUpdateTypes[i].updKind = stateUpdKind(i)
		}
	}
}

func stateUpdateDefaultNoArgPrepare(_ *Slot, stateUpdate *StateUpdate) {
	fn := stateUpdate.param1.(StepPrepareFunc)
	if fn != nil {
		fn()
	}
}

func stateUpdateDefaultVerifyNoArgFn(u interface{}) {
	runtime.KeepAlive(u.(StepPrepareFunc))
}

func stateUpdateDefaultVerifyError(u interface{}) {
	err := u.(error)
	if err == nil {
		panic("illegal value")
	}
}

func stateUpdateDefaultJump(slot *Slot, stateUpdate StateUpdate, worker FixedSlotWorker, sd *StepDeclaration) (isAvailable bool, err error) {
	m := slot.machine
	slot.setNextStep(stateUpdate.step, sd)
	m.updateSlotQueue(slot, worker, activateSlot)
	return true, nil
}

func stateUpdateDefaultStop(slot *Slot, _ StateUpdate, worker FixedSlotWorker, _ *StepDeclaration) (isAvailable bool, err error) {
	m := slot.machine

	if slot.hasSubroutine() {
		slot.applySubroutineStop(nil, worker.asDetachable())
		m.updateSlotQueue(slot, worker, activateSlot)
		return true, nil
	}

	err = slot.callFinalizeOnce(worker.asDetachable(), nil)
	// recycleSlot can handle both in-place and off-place updates
	m.recycleSlot(slot, worker, err)

	return false, nil
}
