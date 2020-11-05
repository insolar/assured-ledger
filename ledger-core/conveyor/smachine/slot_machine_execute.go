// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

/* -------------- Methods to run state machines --------------- */

func (m *SlotMachine) RunToStop(worker AttachableSlotWorker, signal *synckit.SignalVersion) {
	m.Stop()
	worker.AttachTo(m, signal, uint32(m.config.ScanCountLimit), func(worker AttachedSlotWorker) {
		for !m.syncQueue.IsInactive() && !worker.HasSignal() {
			// TODO this causes busy-wait when slots are busy/locked
			m.ScanOnce(ScanDefault, worker)
		}
	})
}

func (m *SlotMachine) ScanNested(outerCtx ExecutionContext, scanMode ScanMode,
	loopLimit uint32, worker AttachableSlotWorker,
) (repeatNow bool, nextPollTime time.Time) {
	if loopLimit == 0 {
		loopLimit = uint32(m.config.ScanCountLimit)
	}
	ec := outerCtx.(*executionContext)

	worker.AttachAsNested(m, ec.w, loopLimit, func(worker AttachedSlotWorker) {
		repeatNow, nextPollTime = m.ScanOnce(scanMode, worker)
	})
	return repeatNow, nextPollTime
}

func (m *SlotMachine) ScanOnce(scanMode ScanMode, worker AttachedSlotWorker) (repeatNow bool, nextPollTime time.Time) {
	status := m.syncQueue.GetStatus()
	if status == SlotMachineInactive {
		return false, time.Time{}
	}

	scanTime := time.Now()
	m.beforeScan(scanTime)
	currentScanNo := uint32(0)

	switch {
	case m.config.BoostNewSlotDuration > 0:
		m.boostPermitLatest = m.boostPermitLatest.reuseOrNew(scanTime)
		switch m.boostPermitEarliest {
		case nil:
			m.boostPermitEarliest = m.boostPermitLatest
		case m.boostPermitLatest:
			// reuse, no need to check
		default:
			m.boostPermitEarliest = m.boostPermitEarliest.discardOlderThan(scanTime.Add(-m.config.BoostNewSlotDuration))
			if m.boostPermitEarliest == nil {
				panic("unexpected")
			}
		}
	case m.boostPermitLatest != nil:
		panic("unexpected")
	}

	switch {
	case m.machineStartedAt.IsZero():
		m.machineStartedAt = scanTime
		fallthrough
	case scanMode == ScanEventsOnly:
		// no scans
		currentScanNo = m.getScanCount()

	case !m.workingSlots.IsEmpty():
		// we were interrupted
		currentScanNo = m.getScanCount()

		if scanMode != ScanPriorityOnly && m.nonBoostedCount >= uint32(m.config.ScanCountLimit) {
			m.nonBoostedCount = 0
			m.workingSlots.PrependAll(&m.boostedSlots)
		}

		if m.nonPriorityCount >= uint32(m.config.ScanCountLimit) {
			m.nonPriorityCount = 0
			m.workingSlots.PrependAll(&m.prioritySlots)
		}

	default:
		currentScanNo = m.incScanCount()

		m.hotWaitOnly = true
		m.scanWakeUpAt = time.Time{}

		m.nonPriorityCount = 0
		m.nonBoostedCount = 0
		m.workingSlots.AppendAll(&m.prioritySlots)

		if scanMode != ScanPriorityOnly {
			m.workingSlots.AppendAll(&m.boostedSlots)
			m.workingSlots.AppendAll(&m.activeSlots)
		}

		m.pollingSlots.FilterOut(scanTime, m.workingSlots.AppendAll)
	}
	m.pollingSlots.PrepareFor(scanTime.Add(m.config.PollingPeriod).Truncate(m.config.PollingTruncate))

	if status == SlotMachineStopping {
		return m.stopAll(worker), time.Time{}
	}

	repeatNow = m.syncQueue.ProcessUpdates(worker.AsFixedSlotWorker())
	hasUpdates, hasSignal, wasDetached := m.syncQueue.ProcessCallbacks(worker)
	if hasUpdates {
		repeatNow = true
	}

	if scanMode != ScanEventsOnly && !hasSignal && !wasDetached {
		m.executeWorkingSlots(currentScanNo, scanMode == ScanPriorityOnly, worker)
		if !m.workingSlots.IsEmpty() {
			repeatNow = true
		}
	}

	repeatNow = repeatNow || !m.hotWaitOnly
	return repeatNow, minTime(m.scanWakeUpAt, m.pollingSlots.GetNearestPollTime())
}

func (m *SlotMachine) beforeScan(scanTime time.Time) {
	if m.machineStartedAt.IsZero() {
		m.machineStartedAt = scanTime
	}
	m.scanStartedAt = scanTime
}

func (m *SlotMachine) stopAll(worker AttachedSlotWorker) (repeatNow bool) {
	fw := worker.AsFixedSlotWorker()
	clean := m.slotPool.ScanAndCleanup(true, func(slot *Slot) {
		m._cleanupSlot(slot, fw, nil)
	}, func(slots []Slot) (isPageEmptyOrWeak, hasWeakSlots bool) {
		return m.stopPage(slots, fw)
	})

	hasUpdates := m.syncQueue.ProcessUpdates(fw)
	hasCallbacks, _, _ := m.syncQueue.ProcessCallbacks(worker)

	if hasUpdates || hasCallbacks || !clean || !m.syncQueue.CleanupDetachQueues() || !m.slotPool.IsEmpty() {
		return true
	}

	m.syncQueue.SetInactive()

	// unsets Slots' reference to SlotMachine to stop retention of SlotMachine by SlotLinks
	m.slotPool.cleanupEmpty()

	return false
}

func (m *SlotMachine) executeWorkingSlots(currentScanNo uint32, priorityOnly bool, worker AttachedSlotWorker) {
	limit := m.config.ScanCountLimit
	for i := 0; i < limit; i++ {
		currentSlot := m.workingSlots.First()
		if currentSlot == nil {
			return
		}
		loopLimit := 1 + ((limit - i) / m.workingSlots.Count())

		// TODO here there may be a collision when shared data is accessed
		// slot should be skipped then
		prevStepNo := currentSlot.startWorking(currentScanNo) // its counterpart is in slotPostExecution()
		currentSlot.removeFromQueue()

		switch {
		case currentSlot.isExecPriority():
			// execute anyway
		case priorityOnly:
			// skip non-priority by putting them back to queues
			if currentSlot.isBoosted() {
				m.boostedSlots.AddLast(currentSlot)
			} else {
				m.activeSlots.AddLast(currentSlot)
			}
			currentSlot.stopWorking()
			continue

		case !currentSlot.isBoosted():
			m.nonBoostedCount++
			fallthrough
		default:
			m.nonPriorityCount++
		}

		stopNow, loopIncrement := m._executeSlot(currentSlot, prevStepNo, worker, loopLimit)
		if stopNow {
			return
		}
		i += loopIncrement
	}
}

func (m *SlotMachine) _executeSlot(slot *Slot, prevStepNo uint32, worker AttachedSlotWorker, loopLimit int) (hasSignal bool, loopCount int) {

	slot.touchAfterInactive()

	if dep := slot.dependency; dep != nil && dep.IsReleaseOnWorking() {
		released := slot._releaseAllDependency()
		m.activateDependants(released, slot.NewLink(), worker.AsFixedSlotWorker())
	}
	slot.slotFlags &^= slotWokenUp|slotPriorityChanged
	postFlags := postExecFlags(0)

	var stateUpdate StateUpdate

	wasDetached := worker.DetachableCall(func(worker DetachableSlotWorker) {
		for ; loopCount < loopLimit; loopCount++ {
			canLoop := false
			canLoop, hasSignal = worker.CanLoopOrHasSignal(loopCount)
			if !canLoop || hasSignal {
				if loopCount == 0 {
					// a very special update type, not to be used anywhere else
					stateUpdate = StateUpdate{updKind: uint8(stateUpdInternalRepeatNow)}
				} else {
					stateUpdate = newStateUpdateTemplate(updCtxExec, 0, stateUpdRepeat).newUint(0)
				}
				return
			}

			var asyncCnt uint16
			var sut StateUpdateType

			if slot.slotFlags & slotStepSuspendMigrate != 0 {
				slot.slotFlags &^= slotStepSuspendMigrate
				postFlags |= wasMigrateSuspended
			}

			ec := executionContext{slotContext: slotContext{s: slot, w: worker}}
			stateUpdate, sut, asyncCnt = ec.executeNextStep()

			slot.addAsyncCount(asyncCnt)
			prevStepDecl := stepToDecl(slot.step, slot.stepDecl)

			switch {
			case !sut.ShortLoop(slot, stateUpdate, uint32(loopCount)):
				return
			case !slot.canMigrateWorking(prevStepNo, true):
				// don't short-loop no-migrate cases to avoid increase of their duration
				return
			}

			switch {
			case !slot.needsReleaseOnStepping(prevStepNo):
				//
			case worker.NonDetachableCall(func(worker FixedSlotWorker) {
				// MUST match SlotMachine.stopSlotWorking
				released := slot._releaseAllDependency()
				link := slot.NewLink()
				m.activateDependants(released, link, worker)
			}):
			default:
				// we need to release, but we were unable to synchronize with SlotMachine
				// cant' short-loop further
				return
			}
			_, prevStepNo, _ = slot._getState()

			slot.logShortLoopUpdate(stateUpdate, prevStepDecl, slot.touchAfterActive())
		}
	})

	if wasDetached {
		// MUST NOT apply any changes in the current routine, as it is no more safe to update queues
		m.asyncPostSlotExecution(slot, stateUpdate, prevStepNo, postFlags)
		return true, loopCount
	}

	hasAsync := m.slotPostExecution(slot, stateUpdate, worker.AsFixedSlotWorker(), prevStepNo, postFlags)
	if hasAsync && !hasSignal {
		_, hasSignal, wasDetached = m.syncQueue.ProcessCallbacks(worker)
		return hasSignal || wasDetached, loopCount
	}
	return hasSignal, loopCount
}

func (m *SlotMachine) _executeSlotInitByCreator(slot *Slot, postInitFn PostInitFunc, worker DetachableSlotWorker) {

	slot.ensureInitializing()
	m._boostNewSlot(slot)

	slot.touchFirstTime() // starts active interval without inactive

	ec := executionContext{slotContext: slotContext{s: slot, w: worker}}
	stateUpdate, _, asyncCnt := ec.executeNextStep()
	slot.addAsyncCount(asyncCnt)

	defer func() {
		if !worker.NonDetachableCall(func(worker FixedSlotWorker) {
			m.slotPostExecution(slot, stateUpdate, worker, 0, wasInlineExec)
		}) {
			m.asyncPostSlotExecution(slot, stateUpdate, 0, wasInlineExec)
		}
	}()

	switch stateUpdKind(stateUpdate.updKind) {
	case stateUpdStop, stateUpdError, stateUpdPanic:
		// init has failed, so there is no reason to call postInitFn
	default:
		postInitFn()
	}
}

type postExecFlags uint8

const (
	wasAsyncExec postExecFlags = 1 << iota
	wasInlineExec
	wasMigrateSuspended
)

func (m *SlotMachine) slotPostExecution(slot *Slot, stateUpdate StateUpdate, worker FixedSlotWorker,
	prevStepNo uint32, flags postExecFlags) (hasAsync bool) {

	wasAsync := flags&wasAsyncExec != 0
	if wasAsync {
		slot.touchAfterAsync()
	}

	slot.logStepUpdate(stateUpdate, flags, slot.touchAfterActive())

	slot.updateBoostFlag()

	if !stateUpdate.IsZero() && !m.applyStateUpdate(slot, stateUpdate, worker) {
		return false
	}

	if slot.canMigrateWorking(prevStepNo, flags&(wasAsyncExec|wasMigrateSuspended) != 0) {
		_, migrateCount := m.getScanAndMigrateCounts()
		if _, isAvailable := m._migrateSlot(migrateCount, slot, worker); !isAvailable {
			return false
		}
	}

	hasAsync = wasAsync || slot.hasAsyncOrBargeIn()
	m.stopSlotWorking(slot, prevStepNo, worker)
	return hasAsync
}

func (m *SlotMachine) applyAsyncCallback(link SlotLink, inlineFlags postExecFlags, worker DetachableSlotWorker,
	callbackFn func(*Slot, DetachableSlotWorker, error) StateUpdate, prevErr error,
) bool {
	if !m._canCallback(link) {
		return true
	}
	if worker.IsZero() {
		if m.IsActive() {
			step, _ := link.GetStepLink()
			m.logInternal(step, "async detachment retry limit exceeded", nil)
		}
		return true
	}

	slot, isStarted, prevStepNo := link.tryStartWorking()
	if !isStarted {
		return false
	}
	var stateUpdate StateUpdate
	func() {
		defer func() {
			stateUpdate = recoverSlotPanicAsUpdate(stateUpdate, "async callback panic", recover(), prevErr, AsyncCallArea)
		}()
		if callbackFn != nil {
			stateUpdate = callbackFn(slot, worker, prevErr)
		}
	}()

	if worker.NonDetachableCall(func(worker FixedSlotWorker) {
		slot.touchAfterAsync()
		m.slotPostExecution(slot, stateUpdate, worker, prevStepNo, inlineFlags)
	}) {
		m.syncQueue.ProcessDetachQueue(link, worker)
	} else {
		m.asyncPostSlotExecution(slot, stateUpdate, prevStepNo, wasAsyncExec)
	}

	return true
}

func (m *SlotMachine) queueAsyncCallback(link SlotLink,
	callbackFn func(*Slot, DetachableSlotWorker, error) StateUpdate, prevErr error) bool {

	if callbackFn == nil && prevErr == nil || !m._canCallback(link) {
		return false
	}

	return m.syncQueue.AddAsyncCallback(link, func(link SlotLink, worker DetachableSlotWorker) (isDone bool) {
		return m.applyAsyncCallback(link, wasAsyncExec, worker, callbackFn, prevErr)
	})
}

func (m *SlotMachine) ensureLocal(s *Slot) {
	// this is safe when s belongs to this m
	// otherwise it shouldn't be here
	if s == nil || s.machine != m {
		panic(throw.IllegalState())
	}
}

func (m *SlotMachine) _canCallback(link SlotLink) bool {
	m.ensureLocal(link.s)
	return link.IsValid()
}

func (m *SlotMachine) asyncPostSlotExecution(s *Slot, stateUpdate StateUpdate, prevStepNo uint32, flags postExecFlags) {
	m.syncQueue.AddAsyncUpdate(s.NewLink(), func(link SlotLink, worker FixedSlotWorker) {
		if !link.IsValid() {
			return
		}
		slot := link.s
		if m.slotPostExecution(slot, stateUpdate, worker, prevStepNo, flags|wasAsyncExec) {
			m.syncQueue.FlushSlotDetachQueue(link)
		}
	})
}
