// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

/* -------------- Methods to run state machines --------------- */

func (m *SlotMachine) RunToStop(worker AttachableSlotWorker, signal *synckit.SignalVersion) {
	m.Stop()
	worker.AttachTo(m, signal, uint32(m.config.ScanCountLimit), func(worker AttachedSlotWorker) {
		for !m.syncQueue.IsInactive() && !worker.HasSignal() {
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

	repeatNow = m.syncQueue.ProcessUpdates(worker)
	hasUpdates, hasSignal, wasDetached := m.syncQueue.ProcessCallbacks(worker)
	if hasUpdates {
		repeatNow = true
	}

	if scanMode != ScanEventsOnly && !hasSignal && !wasDetached {
		m.executeWorkingSlots(currentScanNo, scanMode == ScanPriorityOnly, worker)
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
	clean := m.slotPool.ScanAndCleanup(true, worker, m.recycleSlot, m.stopPage)
	hasUpdates := m.syncQueue.ProcessUpdates(worker)
	hasCallbacks, _, _ := m.syncQueue.ProcessCallbacks(worker)

	if hasUpdates || hasCallbacks || !clean || !m.syncQueue.CleanupDetachQueues() || !m.slotPool.IsEmpty() {
		return true
	}

	m.syncQueue.SetInactive()

	// unsets Slots' reference to SlotMachine to stop retention of SlotMachine by SlotLinks
	m.slotPool._cleanupEmpty()

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

		prevStepNo := currentSlot.startWorking(currentScanNo) // its counterpart is in slotPostExecution()
		currentSlot.removeFromQueue()

		switch {
		case currentSlot.isPriority():
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

	inactivityNano := slot.touch(time.Now().UnixNano())

	if dep := slot.dependency; dep != nil && dep.IsReleaseOnWorking() {
		released := slot._releaseAllDependency()
		m.activateDependants(released, slot.NewLink(), worker)
	}
	slot.slotFlags &^= slotWokenUp

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

			slot.slotFlags &^= slotStepSuspendMigrate
			ec := executionContext{slotContext: slotContext{s: slot, w: worker}}
			stateUpdate, sut, asyncCnt = ec.executeNextStep()

			slot.addAsyncCount(asyncCnt)
			switch {
			case !sut.ShortLoop(slot, stateUpdate, uint32(loopCount)):
				return
			case !slot.canMigrateWorking(prevStepNo, true):
				// don't short-loop no-migrate cases to avoid increase of their duration
				return
			}
			if !slot.needsReleaseOnStepping(prevStepNo) {
				_, prevStepNo, _ = slot._getState()
				continue
			}
			if !worker.NonDetachableCall(func(worker FixedSlotWorker) {
				// MUST match SlotMachine.stopSlotWorking
				released := slot._releaseAllDependency()
				link := slot.NewLink()
				m.activateDependants(released, link, worker)

				_, prevStepNo, _ = slot._getState()
			}) {
				return
			}
		}
	})

	if wasDetached {
		// MUST NOT apply any changes in the current routine, as it is no more safe to update queues
		m.asyncPostSlotExecution(slot, stateUpdate, prevStepNo, inactivityNano)
		return true, loopCount
	}

	hasAsync := m.slotPostExecution(slot, stateUpdate, worker, prevStepNo, false, inactivityNano)
	if hasAsync && !hasSignal {
		_, hasSignal, wasDetached = m.syncQueue.ProcessCallbacks(worker)
		return hasSignal || wasDetached, loopCount
	}
	return hasSignal, loopCount
}

const durationUnknownNano = time.Duration(1)
const durationNotApplicableNano = time.Duration(0)

func (m *SlotMachine) _executeSlotInitByCreator(slot *Slot, postInitFn PostInitFunc, worker DetachableSlotWorker) {

	slot.ensureInitializing()
	m._boostNewSlot(slot)

	slot.touch(time.Now().UnixNano())

	ec := executionContext{slotContext: slotContext{s: slot, w: worker}}
	stateUpdate, _, asyncCnt := ec.executeNextStep()
	slot.addAsyncCount(asyncCnt)

	defer func() {
		if !worker.NonDetachableCall(func(worker FixedSlotWorker) {
			m.slotPostExecution(slot, stateUpdate, worker, 0, false, durationUnknownNano)
		}) {
			m.asyncPostSlotExecution(slot, stateUpdate, 0, durationUnknownNano)
		}
	}()

	switch stateUpdKind(stateUpdate.updKind) {
	case stateUpdStop, stateUpdError, stateUpdPanic:
		// init has failed, so there is no reason to call postInitFn
	default:
		postInitFn()
	}
}

func (m *SlotMachine) slotPostExecution(slot *Slot, stateUpdate StateUpdate, worker FixedSlotWorker,
	prevStepNo uint32, wasAsync bool, inactivityNano time.Duration) (hasAsync bool) {

	activityNano := durationNotApplicableNano
	if !wasAsync && inactivityNano > durationNotApplicableNano {
		activityNano = slot.touch(time.Now().UnixNano())
	}

	slot.logStepUpdate(stateUpdate, wasAsync, inactivityNano, activityNano)

	slot.updateBoostFlag()

	if !stateUpdate.IsZero() && !m.applyStateUpdate(slot, stateUpdate, worker) {
		return false
	}

	if slot.canMigrateWorking(prevStepNo, wasAsync) {
		_, migrateCount := m.getScanAndMigrateCounts()
		if _, isAvailable := m._migrateSlot(migrateCount, slot, worker); !isAvailable {
			return false
		}
	}

	hasAsync = wasAsync || slot.hasAsyncOrBargeIn()
	m.stopSlotWorking(slot, prevStepNo, worker)
	return hasAsync
}

func (m *SlotMachine) applyAsyncCallback(link SlotLink, worker DetachableSlotWorker,
	callbackFn func(*Slot, DetachableSlotWorker, error) StateUpdate, prevErr error,
) bool {
	if !m._canCallback(link) {
		return true
	}
	if worker == nil {
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
		m.slotPostExecution(slot, stateUpdate, worker, prevStepNo, true, durationNotApplicableNano)
	}) {
		m.syncQueue.ProcessDetachQueue(link, worker)
	} else {
		m.asyncPostSlotExecution(slot, stateUpdate, prevStepNo, durationNotApplicableNano)
	}

	return true
}

func (m *SlotMachine) queueAsyncCallback(link SlotLink,
	callbackFn func(*Slot, DetachableSlotWorker, error) StateUpdate, prevErr error) bool {

	if callbackFn == nil && prevErr == nil || !m._canCallback(link) {
		return false
	}

	return m.syncQueue.AddAsyncCallback(link, func(link SlotLink, worker DetachableSlotWorker) (isDone bool) {
		return m.applyAsyncCallback(link, worker, callbackFn, prevErr)
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

func (m *SlotMachine) asyncPostSlotExecution(s *Slot, stateUpdate StateUpdate, prevStepNo uint32, inactivityNano time.Duration) {
	m.syncQueue.AddAsyncUpdate(s.NewLink(), func(link SlotLink, worker FixedSlotWorker) {
		if !link.IsValid() {
			return
		}
		slot := link.s
		if m.slotPostExecution(slot, stateUpdate, worker, prevStepNo, true, inactivityNano) {
			m.syncQueue.FlushSlotDetachQueue(link)
		}
	})
}
