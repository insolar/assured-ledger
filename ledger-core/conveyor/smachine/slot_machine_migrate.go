package smachine

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

/* -- Methods to migrate slots ------------------------------ */

func (m *SlotMachine) TryMigrateNested(outerCtx ExecutionContext) bool {
	ec := outerCtx.(*executionContext)
	return ec.w.NonDetachableCall(m.migrate)
}

func (m *SlotMachine) MigrateNested(outerCtx MigrationContext) {
	mc := outerCtx.(*migrationContext)
	m.migrate(mc.fixedWorker)
}

func (m *SlotMachine) migrate(worker FixedSlotWorker) {
	m.migrateWithBefore(worker, nil)
}

func (m *SlotMachine) migrateWithBefore(worker FixedSlotWorker, beforeFn func()) {
	migrateCount := m.incMigrateCount()

	if beforeFn != nil {
		beforeFn()
	}

	for _, fn := range m.migrators {
		fn(migrateCount)
	}

	m.slotPool.ScanAndCleanup(m.config.CleanupWeakOnMigrate, func(slot *Slot) {
		m._cleanupSlot(slot, worker, nil)
	}, func(slotPage []Slot) (isPageEmptyOrWeak, hasWeakSlots bool) {
		return m.migratePage(migrateCount, slotPage, worker)
	})

	m.syncQueue.CleanupDetachQueues()
}

func (m *SlotMachine) AddMigrationCallback(fn MigrationFunc) {
	if fn == nil {
		panic("illegal value")
	}
	m.migrators = append(m.migrators, fn)
}

func (m *SlotMachine) migratePage(migrateCount uint32, slotPage []Slot, worker FixedSlotWorker) (isPageEmptyOrWeak, hasWeakSlots bool) {
	isPageEmptyOrWeak = true
	hasWeakSlots = false
	for i := range slotPage {
		isSlotEmptyOrWeak, isSlotAvailable := m.migrateSlot(migrateCount, &slotPage[i], worker)
		switch {
		case !isSlotEmptyOrWeak:
			isPageEmptyOrWeak = false
		case isSlotAvailable:
			hasWeakSlots = true
		}
	}
	return isPageEmptyOrWeak, hasWeakSlots
}

func (m *SlotMachine) migrateSlot(migrateCount uint32, slot *Slot, w FixedSlotWorker) (isEmptyOrWeak, isAvailable bool) {
	isEmpty, isStarted, prevStepNo := slot.tryStartMigrate()
	if !isStarted {
		return isEmpty, false
	}
	if slot.slotFlags&slotStepSuspendMigrate != 0 {
		m.stopSlotWorking(slot, prevStepNo, w)
		return false, false
	}

	isEmptyOrWeak, isAvailable = m._migrateSlot(migrateCount, slot, w)
	if isAvailable {
		m.stopSlotWorking(slot, prevStepNo, w)
	}
	return isEmptyOrWeak, isAvailable
}

func (m *SlotMachine) _migrateSlot(lastMigrationCount uint32, slot *Slot, worker FixedSlotWorker) (isEmptyOrWeak, isAvailable bool) {

	slot.touchAfterInactive()

	var pendingUpdate StateUpdate

	for delta := lastMigrationCount - slot.migrationCount; delta > 0; {
		if migrateFn, ms := slot.getMigration(), slot.stateStack; migrateFn != nil || ms != nil && ms.hasMigrates {
			slot.runShadowMigrate(1)

			skipMultiple := true

			for level := 0; ; level++ {
				if migrateFn != nil {
					mc := migrationContext{
						slotContext: slotContext{s: slot, w: worker.asDetachable()},
						fixedWorker: worker,
						callerSM: level > 0,
					}
					stateUpdate, skipAll := mc.executeMigration(migrateFn)

					slot.logStepMigrate(stateUpdate, migrateFn, slot.touchAfterActive())

					switch {
					case !typeOfStateUpdate(stateUpdate).IsSubroutineSafe():
						pendingUpdate = stateUpdate
						skipMultiple = skipAll
						switch {
						case level == 0:
							break
						case ms != nil:
							slot._popTillSubroutine(ms.childMarker, worker.asDetachable())
						default:
							slot._popTillSubroutine(subroutineMarker{}, worker.asDetachable())
						}
					case !m.applyStateUpdate(slot, stateUpdate, worker):
						panic(throw.IllegalState())
						// return true, false
					case !skipAll:
						skipMultiple = false
					}
				}
				if ms == nil || !ms.hasMigrates {
					break
				}
				migrateFn = ms.stackMigrate
				ms = ms.stateStack
			}
			slot.migrationCount++
			delta--
			if delta == 0 {
				break
			}
			if !skipMultiple {
				continue
			}
		}
		if delta > 0 {
			slot.runShadowMigrate(delta)
		}
		slot.migrationCount = lastMigrationCount
		break
	}

	if !m.applyStateUpdate(slot, pendingUpdate, worker) {
		return true, false
	}

	return slot.step.Flags&StepWeak != 0, true
}
