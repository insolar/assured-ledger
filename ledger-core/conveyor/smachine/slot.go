package smachine

import (
	"context"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Slot struct {
	idAndStep uint64 // atomic access

	// machine can ONLY be accessed directly while is a part of active context / fixed worker
	// if any doubt - use getMachine() instead
	// atomic access, set on first use, unset only after SlotMachine termination
	machine *SlotMachine

	/* -----------------------------------
	   Slot fields to support processing queues
	   -----------------------------------
	   SYNC: these portion of slot can be accessed:
		- if queue is assigned - by the goroutine owning the queue's head
	    - if queue is unassigned - by the goroutine of the machine
	*/
	prevInQueue *Slot
	nextInQueue *Slot
	queue       *QueueHead

	/* SYNC: this portion of slot can ONLY be accessed by:
	- the same goroutine that has set BUSY
	- or for non-BUSY - the goroutine of the machine
	*/
	slotData
}

// stateMachineData contains details about specific StateMachine instance
type stateMachineData struct {
	declaration     StateMachineHelper
	shadowMigrate   ShadowMigrateFunc              // nolint runs for all subroutines
	stepLogger      StepLogger                     // nolint
	defTerminate    internalTerminationHandlerFunc // nolint
	defFinalize     FinalizeFunc				   // nolint
	defMigrate      MigrateFunc                    // nolint
	defErrorHandler ErrorHandlerFunc               // nolint
	defFlags        StepFlags                      // nolint

	// DO NOT modify content of this map
	inheritable map[string]interface{} // nolint

	stateStack *stackedStateMachineData
}

type stackedStateMachineData struct {
	stateMachineData
	stackMigrate MigrateFunc
	returnFn     SubroutineExitFunc
	childMarker  subroutineMarker
	copyAliases  *slotAliases

	cleanupMode SubroutineCleanupMode
	hasMigrates bool
}

// nolint
type slotData struct {
	parent SlotLink
	ctx    context.Context

	stateMachineData

	boost    *boostPermit
	step     SlotStep
	stepDecl *StepDeclaration

	dependency SlotDependency
	defResult  interface{}

	touchLast int64
	touchAux  time.Duration

	migrationCount uint32 // can be wrapped on overflow
	asyncCallCount uint16 // pending calls, overflow panics
	lastWorkScan   uint8  // to check if a slot was executed in this cycle
	slotFlags      slotFlags
}

type slotFlags uint8

const (
	slotWokenUp slotFlags = 1 << iota
	slotHadBargeIn
	slotHadAliases
	slotHadAsync // purely diagnostic
	slotIsTracing
	slotIsBoosted
	slotPriorityChanged
	slotStepSuspendMigrate
)

type SlotDependency interface {
	IsReleaseOnWorking() bool
	IsReleaseOnStepping() bool
	ReleaseAll() ([]PostponedDependency, []StepLink)
}

type PostponedDependency interface {
	PostponedActivate(appendTo []StepLink) []StepLink
}

const (
	slotFlagBusyShift = 32 + iota
	stepIncrementShift
)

const stepIncrement uint64 = 1 << stepIncrementShift
const slotFlagBusy uint64 = 1 << slotFlagBusyShift
const slotIDMask = slotFlagBusy - 1

// see also numberOfReservedSteps

/*
	Slot and step numbers are interpreted together with isBusy flag:
	[step] [busy] [slotID]
	[  0 ] [  0 ] [    0 ] slot is unused and is owned by a slot pool
	[  N ] [  0 ] [   >0 ] slot is in use and is not owned by a slot pool
	[  N ] [  1 ] [    N ] slot is in use and is not owned by a slot pool

	[  0 ] [  0 ] [   >0 ] illegal/reserved state
	[  0 ] [  1 ] [   >0 ] illegal/reserved state
	[ >0 ] [  0 ] [    0 ] illegal/reserved state


	[ >0 ] [  0 ] [   >0 ] slot is allocated and idle (only slot machine can access it); includes:
	 [ 1 ] [  0 ] [   >0 ] slot is constructed and awaits initialization
	 [>1 ] [  0 ] [   >0 ] slot is operational and idle

	[ >0 ] [  1 ] [   >0 ] slot is allocated and busy (only busy-holder can access it, can't be migrated); includes:
	 [ 1 ] [  1 ] [   >0 ] slot is allocated and is being constructed/initialized
	 [>1 ] [  1 ] [   >0 ] initialized slot is busy / working
	 [ 1 ] [  1 ] [    0 ] uninitialized slot is being finalized
	 [>1 ] [  1 ] [    0 ] initialized slot is being finalized


	Usual slot lifecycle is:
	[  0 ] [  0 ] [    0 ] slot is unused and is owned by a slot pool
	[  1 ] [  1 ] [   >0 ] slot is allocated and is being constructed/initialized (can't be migrated)
	[  1 ] [  0 ] [   >0 ] slot is constructed and awaits initialization (can't be migrated)
    [ >1 ] [  N ] [   >0 ] slot is operational (idle / working)
    [ >1 ] [  1 ] [    0 ] slot is being finalized (can't be migrated)
	[  0 ] [  0 ] [    0 ] slot is unused and is owned by a slot pool

	Step number is a cyclic incrementing counter with reserved values:
	= 0 - slot is is just allocated (+busy flag) or is not used by a state machine
	= 1 - slot is initializing, can only appear once for a state machine
	On overflow, step will change to =numberOfReservedSteps
*/

func (s *Slot) _getState() (id SlotID, step uint32, isBusy bool) {
	v := atomic.LoadUint64(&s.idAndStep)
	return SlotID(v), uint32(v >> stepIncrementShift), v&slotFlagBusy != 0
}

func (s *Slot) GetState() (id SlotID, step uint32, isBusy bool) {
	v := atomic.LoadUint64(&s.idAndStep)
	if SlotID(v).IsUnknown() || v <= slotFlagBusy {
		panic("illegal state")
	}
	return SlotID(v), uint32(v >> stepIncrementShift), v&slotFlagBusy != 0
}

func (s *Slot) GetSlotID() SlotID {
	v := atomic.LoadUint64(&s.idAndStep)
	if SlotID(v).IsUnknown() || v <= slotFlagBusy {
		panic("illegal state")
	}
	return SlotID(v)
}

func (s *Slot) getMachine() *SlotMachine {
	return (*SlotMachine)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&s.machine))))
}

func (s *Slot) unsetMachine() bool {
	return atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(&s.machine)), nil) != nil
}

func (s *Slot) invalidateSlotID() {
	for {
		v := atomic.LoadUint64(&s.idAndStep)
		if SlotID(v).IsUnknown() || v <= slotFlagBusy {
			panic("illegal state")
		}
		if atomic.CompareAndSwapUint64(&s.idAndStep, v, slotFlagBusy|v&^slotIDMask) {
			return
		}
	}
}

func (s *Slot) isEmpty() bool {
	return atomic.LoadUint64(&s.idAndStep) == 0
}

func (s *Slot) isBusy() bool {
	return atomic.LoadUint64(&s.idAndStep)&slotFlagBusy != 0
}

func (s *Slot) isInitializing() bool {
	v := atomic.LoadUint64(&s.idAndStep)
	return v&^(slotFlagBusy-1) == slotFlagBusy|stepIncrement
}

func (s *Slot) ensureInitializing() {
	if !s.isInitializing() {
		panic("illegal state")
	}
}

func (s *Slot) _slotAllocated(id SlotID) {
	if id == 0 {
		atomic.StoreUint64(&s.idAndStep, slotFlagBusy)
	} else {
		atomic.StoreUint64(&s.idAndStep, uint64(id)|stepIncrement|slotFlagBusy)
	}
}

// nolint:unused
func (s *Slot) _trySetFlag(f uint64) (bool, uint64) {
	for {
		v := atomic.LoadUint64(&s.idAndStep)
		if v&f != 0 {
			return false, 0
		}

		if atomic.CompareAndSwapUint64(&s.idAndStep, v, v|f) {
			return true, v
		}
	}
}

// nolint:unused
func (s *Slot) _setFlag(f uint64) uint64 {
	ok, v := s._trySetFlag(f)
	if !ok {
		panic("illegal state")
	}
	return v
}

// nolint:unused
func (s *Slot) _unsetFlag(f uint64) uint64 {
	for {
		v := atomic.LoadUint64(&s.idAndStep)
		if v&f == 0 {
			panic("illegal state")
		}

		if atomic.CompareAndSwapUint64(&s.idAndStep, v, v&^f) {
			return v
		}
	}
}

func (s *Slot) incStep() {
	for {
		v := atomic.LoadUint64(&s.idAndStep)
		if SlotID(v) == 0 {
			panic("illegal state")
		}
		update := v + stepIncrement
		if update < stepIncrement {
			// overflow, skip steps 0 and 1
			update += stepIncrement * numberOfReservedSteps
		}
		if atomic.CompareAndSwapUint64(&s.idAndStep, v, update) {
			return
		}
	}
}

func (s *Slot) _tryStartSlot(minStepNo uint32) (isEmpty, isStarted bool, prevStepNo uint32) {
	for {
		v := atomic.LoadUint64(&s.idAndStep)
		if v == 0 /* isEmpty() */ {
			return true, false, 0
		}

		prevStepNo = uint32(v >> stepIncrementShift)
		if v&slotFlagBusy != 0 /* isBusy() */ || v < slotFlagBusy /* reserved */ || prevStepNo < minStepNo {
			return false, false, prevStepNo
		}

		if atomic.CompareAndSwapUint64(&s.idAndStep, v, v|slotFlagBusy) {
			return false, true, prevStepNo
		}
	}
}

func (s *Slot) _tryStartWithID(slotID SlotID, minStepNo uint32) (isValid, isStarted bool, prevStepNo uint32) { // nolint:unparam
	for {
		v := atomic.LoadUint64(&s.idAndStep)
		if v == 0 /* isEmpty() */ || SlotID(v) != slotID {
			return false, false, 0
		}

		prevStepNo = uint32(v >> stepIncrementShift)
		if v&slotFlagBusy != 0 /* isBusy() */ || v < slotFlagBusy /* reserved */ || prevStepNo < minStepNo {
			return false, false, prevStepNo
		}

		if atomic.CompareAndSwapUint64(&s.idAndStep, v, v|slotFlagBusy) {
			return false, true, prevStepNo
		}
	}
}

func (s *Slot) stopWorking() {
	for {
		v := atomic.LoadUint64(&s.idAndStep)
		if v&slotFlagBusy == 0 {
			panic("illegal state")
		}

		if atomic.CompareAndSwapUint64(&s.idAndStep, v, v&^slotFlagBusy) {
			return
		}
	}
}

func (s *Slot) canMigrateWorking(prevStepNo uint32, migrateIsNeeded bool) bool {
	switch {
	case s.slotFlags&slotStepSuspendMigrate != 0:
		return false
	case prevStepNo > 1:
		return migrateIsNeeded
	case prevStepNo != 1:
		return false
	default:
		return atomic.LoadUint64(&s.idAndStep) >= stepIncrement*numberOfReservedSteps
	}
}

func (s *Slot) tryStartMigrate() (isEmpty, isStarted bool, prevStepNo uint32) {
	isEmpty, isStarted, prevStepNo = s._tryStartSlot(2)
	return
}

func (s *Slot) startWorking(scanNo uint32) uint32 {
	if _, isStarted, prevStepNo := s._tryStartSlot(1); isStarted {
		s.lastWorkScan = uint8(scanNo)
		return prevStepNo
	}
	panic("illegal state")
}

func (s *Slot) isInQueue() bool {
	return s.queue != nil || s.nextInQueue != nil || s.prevInQueue != nil
}

func (s *Slot) ensureNotInQueue() {
	if s.isInQueue() {
		panic("illegal state")
	}
}

func (s *Slot) ensureInQueue() {
	if s.queue == nil || s.nextInQueue == nil || s.prevInQueue == nil {
		panic("illegal state")
	}
}

func (s *Slot) dispose() {
	s.ensureNotInQueue()
	if s.slotData.dependency != nil {
		panic("illegal state")
	}
	atomic.StoreUint64(&s.idAndStep, 0)
	s.slotData = slotData{}
}

func (s *Slot) NewLink() SlotLink {
	id, _, _ := s.GetState()
	return SlotLink{id, s}
}

func (s *Slot) NewStepLink() StepLink {
	id, step, _ := s.GetState()
	return StepLink{SlotLink{id, s}, step}
}

func (s *slotData) isLastScan(scanNo uint32) bool {
	return s.lastWorkScan == uint8(scanNo)
}

func (s *Slot) setNextStep(step SlotStep, stepDecl *StepDeclaration) {
	if step.Transition == nil {
		if step.Flags != 0 || step.Migration != nil {
			panic(throw.IllegalValue())
		}
		// leave as-is
		return
	}
	if stepDecl == nil {
		stepDecl = s.declaration.GetStepDeclaration(step.Transition)
	}

	defFlags := s.defFlags
	if stepDecl != nil {
		defFlags |= stepDecl.Flags
		if step.Migration == nil {
			step.Migration = stepDecl.Migration
		}
		if step.Handler == nil {
			step.Handler = stepDecl.Handler
		}
	}

	if step.Flags&StepResetAllFlags == 0 {
		step.Flags |= defFlags
	} else {
		step.Flags &^= StepResetAllFlags
	}

	if (s.step.Flags & StepPriority) != (step.Flags & StepPriority) {
		s.slotFlags ^= slotPriorityChanged
	}

	s.step = step
	s.stepDecl = stepDecl
	s.incStep()
}

func (s *Slot) removeHeadedQueue() *Slot {
	nextDep, _, _ := s.queue.extractAll(nil)
	s.vacateQueueHead()
	return nextDep
}

// nolint:unused
func (s *Slot) ensureLocal(link SlotLink) {
	switch m := s.getMachine(); {
	case m == nil:
		panic(throw.IllegalState())
	case m != link.getMachine():
		panic(throw.IllegalState())
	}
}

func (s *Slot) isExecPriority() bool {
	return s.step.Flags&StepPriority != 0
}

func (s *Slot) isSyncPriority() bool {
	return s.isExecPriority()
}

func (s *Slot) isSyncBoost() bool {
	return s.step.Flags&StepSyncBoost != 0 || s.isBoosted()
}

func (s *Slot) getMigration() MigrateFunc {
	if s.step.Migration != nil {
		return s.step.Migration
	}
	return s.defMigrate
}

func (s *Slot) getErrorHandler() ErrorHandlerFunc {
	if s.step.Handler != nil {
		return s.step.Handler
	}
	return s.defErrorHandler
}

func (s *Slot) hasAsyncOrBargeIn() bool {
	return s.asyncCallCount > 0 || s.slotFlags&slotHadBargeIn != 0
}

func (s *Slot) addAsyncCount(asyncCnt uint16) {
	if asyncCnt == 0 {
		return
	}
	s.slotFlags |= slotHadAsync
	asyncCnt += s.asyncCallCount
	if asyncCnt <= s.asyncCallCount {
		panic("overflow")
	}
	s.asyncCallCount = asyncCnt
}

func (s *Slot) decAsyncCount() {
	if s.asyncCallCount == 0 {
		panic("underflow")
	}
	s.asyncCallCount--
}

func stepToDecl(step SlotStep, stepDecl *StepDeclaration) StepDeclaration {
	if stepDecl == nil {
		return StepDeclaration{SlotStep: step}
	}
	return StepDeclaration{SlotStep: step, stepDeclExt: stepDecl.stepDeclExt}
}

func (s *Slot) newStepLoggerData(eventType StepLoggerEvent, link StepLink) StepLoggerData {
	flags := StepLoggerFlags(0)
	if s.getStepLogLevel() >= StepLogLevelElevated {
		flags |= StepLoggerElevated
	}

	return StepLoggerData{
		CycleNo:     s.getMachine().getScanCount(),
		StepNo:      link,
		CurrentStep: stepToDecl(s.step, s.stepDecl),
		Declaration: s.declaration,
		EventType:   eventType,
		Flags:       flags,
	}
}

func (s *Slot) logInternal(link StepLink, updateType string, err error) {
	switch {
	case s.stepLogger == nil:
		return
	case err != nil:
		if !s.stepLogger.CanLogEvent(StepLoggerInternal, StepLogLevelError) {
			return
		}
	case !s.stepLogger.CanLogEvent(StepLoggerInternal, s.getStepLogLevel()):
		return
	}

	stepData := s.newStepLoggerData(StepLoggerInternal, link)
	stepData.Error = err

	func() {
		defer func() {
			_ = recover() // we can't fail logInternal() call
		}()
		s.stepLogger.LogInternal(stepData, updateType)
	}()
}

func (s *Slot) logStepError(action ErrorHandlerAction, stateUpdate StateUpdate, area SlotPanicArea, err error) {
	if !s._canLogEvent(StepLoggerUpdate, true) {
		return
	}

	flags := StepLoggerUpdateErrorDefault
	switch action {
	case ErrorHandlerMute:
		flags = StepLoggerUpdateErrorMuted
	case ErrorHandlerRecover, ErrorHandlerRecoverAndWakeUp:
		flags = StepLoggerUpdateErrorRecovered
	case errorHandlerRecoverDenied:
		flags = StepLoggerUpdateErrorRecoveryDenied
	}

	if area.IsDetached() {
		flags |= StepLoggerDetached
	}
	s._logStepUpdate(StepLoggerUpdate, s.touchAfterActive(), stateUpdate,
		func(d *StepLoggerData, _ *StepLoggerUpdateData) {
			d.Flags |= flags
			d.Error = err
		})
}

func (s *Slot) logStepMigrate(stateUpdate StateUpdate, appliedMigrateFn MigrateFunc, timings scheduleTimings) {
	if !s._canLogEvent(StepLoggerMigrate, false) {
		return
	}

	s._logStepUpdate(StepLoggerMigrate, timings, stateUpdate, func(_ *StepLoggerData, upd *StepLoggerUpdateData) {
		upd.AppliedMigrate = appliedMigrateFn
	})
}

func (s *Slot) logStepUpdate(stateUpdate StateUpdate, flags postExecFlags, timings scheduleTimings) {
	if !s._canLogEvent(StepLoggerUpdate, false) {
		return
	}

	s._logStepUpdate(StepLoggerUpdate, timings, stateUpdate, func(d *StepLoggerData, _ *StepLoggerUpdateData) {
		if flags&wasAsyncExec != 0 {
			d.Flags |= StepLoggerDetached
		}
		if flags&wasInlineExec != 0 {
			d.Flags |= StepLoggerInline
		}
	})
}

func (s *Slot) logShortLoopUpdate(stateUpdate StateUpdate, curStep StepDeclaration, timings scheduleTimings) {
	if !s._canLogEvent(StepLoggerUpdate, false) {
		return
	}

	s._logStepUpdate(StepLoggerUpdate, timings, stateUpdate, func(d *StepLoggerData, _ *StepLoggerUpdateData) {
		d.Flags |= StepLoggerShortLoop
		d.CurrentStep = curStep
	})
}

func (s *Slot) _canLogEvent(eventType StepLoggerEvent, hasError bool) bool {
	switch {
	case s.stepLogger == nil:
		return false
	case hasError:
		return s.stepLogger.CanLogEvent(eventType, StepLogLevelError)
	}
	return s.stepLogger.CanLogEvent(eventType, s.getStepLogLevel())
}

func (s *Slot) _logStepUpdate(eventType StepLoggerEvent, timings scheduleTimings,
	stateUpdate StateUpdate, fn func(*StepLoggerData, *StepLoggerUpdateData)) {

	stepData := s.newStepLoggerData(eventType, s.NewStepLink())
	updData := StepLoggerUpdateData{
		InactivityNano: timings.inactivityNano,
		ActivityNano:   timings.activityNano,
	}

	sut, sutName, _ := getStateUpdateTypeAndName(stateUpdate)
	updData.UpdateType = sutName

	if fn != nil {
		fn(&stepData, &updData)
	}

	if nextStep := stateUpdate.step.Transition; nextStep != nil {
		nextDecl := sut.GetStepDeclaration()
		if nextDecl == nil {
			nextDecl = s.declaration.GetStepDeclaration(nextStep)
		}
		updData.NextStep = stepToDecl(stateUpdate.step, nextDecl)
	} else {
		updData.NextStep.SlotStep = stateUpdate.step
	}

	s.stepLogger.LogUpdate(stepData, updData)
}

func (s *Slot) setStepLoggerAfterInit(updateFn StepLoggerUpdateFunc) {
	newStepLogger := updateFn(s.stepLogger, s.getMachine().config.SlotMachineLogger.CreateStepLogger)

	if newStepLogger == nil && s.stepLogger != nil {
		tracerID := s.stepLogger.GetTracerID()
		if len(tracerID) > 0 {
			newStepLogger = StepLoggerStub{TracerID: tracerID}
		}
	}
	s.stepLogger = newStepLogger
}

func (s *Slot) getStepLogLevel() StepLogLevel {
	switch {
	case s.slotFlags&slotIsTracing != 0:
		return StepLogLevelTracing
	case s.step.Flags&StepElevatedLog != 0:
		return StepLogLevelElevated
	default:
		return StepLogLevelDefault
	}
}

func (s *Slot) getAdapterLogging() bool {
	return s.getStepLogLevel() != StepLogLevelDefault || s.getMachine().getAdapterLogging()
}

func (s *Slot) isTracing() bool {
	return s.slotFlags&slotIsTracing != 0
}

func (s *Slot) setTracing(b bool) {
	if b {
		s.slotFlags |= slotIsTracing
	} else {
		s.slotFlags &^= slotIsTracing
	}
}

func (s *Slot) getTracerID() TracerID {
	if s.stepLogger == nil {
		return ""
	}
	return s.stepLogger.GetTracerID()
}

func (s *Slot) updateBoostFlag() {
	if s.slotFlags&slotIsBoosted != 0 && !s.boost.isActive() {
		s.slotFlags &^= slotIsBoosted
		s.boost = nil
	}
}

func (s *Slot) isBoosted() bool {
	return s.slotFlags&slotIsBoosted != 0
}

type scheduleTimings struct {
	inactivityNano, activityNano time.Duration
}

const durationUnknownOrTooShortNano = time.Duration(1)
const durationNotApplicableNano = time.Duration(0)

func (s *Slot) touchAfterInactive() {
	touchedAt := time.Now().UnixNano()
	if s.touchLast > 0 {
		if s.touchAux = time.Duration(touchedAt - s.touchLast); s.touchAux <= 0 {
			s.touchAux = durationUnknownOrTooShortNano
		}
	} else {
		s.touchAux = durationNotApplicableNano
	}
	s.touchLast = touchedAt
}

func (s *Slot) touchAfterAsync() {
	if s.touchLast == 0 {
		return
	}

	touchedAt := time.Now().UnixNano()
	if s.touchAux = time.Duration(touchedAt - s.touchLast); s.touchAux <= 0 {
		s.touchAux = durationUnknownOrTooShortNano
	}
	s.touchLast = 0
}

func (s *Slot) touchFirstTime() {
	s.touchAux = durationNotApplicableNano
	s.touchLast = time.Now().UnixNano()
}

func (s *Slot) touchAfterActive() (timings scheduleTimings) {
	touchedAt := time.Now().UnixNano()

	timings.inactivityNano = s.touchAux
	if s.touchLast > 0 {
		if timings.activityNano = time.Duration(touchedAt - s.touchLast); timings.activityNano <= 0 {
			timings.activityNano = durationUnknownOrTooShortNano
		}
	}
	s.touchAux = durationUnknownOrTooShortNano
	s.touchLast = touchedAt

	return
}

func (s *Slot) runShadowMigrate(migrationDelta uint32) {
	if s.shadowMigrate != nil {
		s.shadowMigrate(s.migrationCount, migrationDelta)
	}

	for ms := s.stateStack; ms != nil && ms.hasMigrates; ms = ms.stateStack {
		if ms.shadowMigrate != nil {
			ms.shadowMigrate(s.migrationCount, migrationDelta)
		}
	}
}

func (s *Slot) _addTerminationCallback(link SlotLink, fn TerminationCallbackFunc) {
	if fn == nil || !link.IsValid() {
		return
	}
	prevTermFn := s.defTerminate
	s.defTerminate = func(data TerminationData, worker FixedSlotWorker) {
		if link.IsValid() {
			if m := link.getActiveMachine(); m != nil {
				resultFn := fn(data.Result, data.Error)
				m.queueAsyncResultCallback(link.GetAnyStepLink(), 0, nil, resultFn, nil, true)
			}
		}

		if prevTermFn != nil {
			prevTermFn(data, worker)
		}
	}
}
