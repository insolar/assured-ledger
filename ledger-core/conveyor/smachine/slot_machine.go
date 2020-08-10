// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type MigrationFunc func(migrationCount uint32)

type SlotMachineConfig struct {
	PollingPeriod        time.Duration
	PollingTruncate      time.Duration
	BoostNewSlotDuration time.Duration
	SlotPageSize         uint16
	ScanCountLimit       int
	CleanupWeakOnMigrate bool

	LogAdapterCalls   bool
	SlotMachineLogger SlotMachineLogger

	SlotIDGenerateFn func() SlotID

	SlotAliasRegistry SlotAliasRegistry
}

type SlotAliasRegistry interface {
	PublishAlias(key interface{}, slot SlotAliasValue) bool
	ReplaceAlias(key interface{}, slot SlotAliasValue)
	UnpublishAlias(key interface{})
	GetPublishedAlias(key interface{}) SlotAliasValue
}

type SlotAliasValue struct {
	Link    SlotLink
	BargeIn BargeInHolder
}

const maxLoopCount = 10000

func NewSlotMachine(config SlotMachineConfig,
	eventCallback, signalCallback func(),
	parentRegistry injector.DependencyRegistry,
) *SlotMachine {
	if config.ScanCountLimit <= 0 || config.ScanCountLimit > maxLoopCount {
		config.ScanCountLimit = maxLoopCount
	}

	m := &SlotMachine{
		config:         config,
		parentRegistry: parentRegistry,
		slotPool:       newSlotPool(config.SlotPageSize, false),
		syncQueue:      newSlotMachineSync(eventCallback, signalCallback),
	}

	if m.config.SlotIDGenerateFn == nil {
		m.config.SlotIDGenerateFn = m._allocateNextSlotID
	}

	m.slotPool.initSlotPool()

	m.activeSlots.initSlotQueue(ActiveSlots)
	m.boostedSlots.initSlotQueue(ActiveSlots)
	m.prioritySlots.initSlotQueue(ActiveSlots)

	m.workingSlots.initSlotQueue(WorkingSlots)

	return m
}

var _ injector.DependencyRegistry = &SlotMachine{}

type SlotMachine struct {
	config     SlotMachineConfig
	lastSlotID SlotID // atomic
	slotPool   SlotPool

	parentRegistry injector.DependencyRegistry
	localRegistry  sync.Map // is used for both dependencies and tracking of dependencies

	machineStartedAt time.Time
	scanStartedAt    time.Time

	boostPermitLatest   *chainedBoostPermit
	boostPermitEarliest *chainedBoostPermit

	scanAndMigrateCounts uint64 // atomic

	migrators []MigrationFunc

	hotWaitOnly  bool      // true when activeSlots & prioritySlots have only slots added by "hot wait"
	scanWakeUpAt time.Time // when all slots are waiting, this is the earliest time requested for wakeup

	prioritySlots    SlotQueue //they are are moved to workingSlots every time when enough non-priority slots are processed
	boostedSlots     SlotQueue
	activeSlots      SlotQueue    //they are are moved to workingSlots on every full Scan
	pollingSlots     PollingQueue //they are are moved to workingSlots on every full Scan when time has passed
	workingSlots     SlotQueue    //slots are currently in processing
	nonPriorityCount uint32       // number of non-priority slots processed since the last replenishment of workingSlots
	nonBoostedCount  uint32       // number of non-priority slots processed since the last replenishment of workingSlots

	syncQueue SlotMachineSync
}

type ScanMode uint8

const (
	ScanDefault ScanMode = iota
	ScanPriorityOnly
	ScanEventsOnly
)

func (m *SlotMachine) IsZero() bool {
	return m.syncQueue.IsZero()
}

func (m *SlotMachine) IsEmpty() bool {
	return m.slotPool.IsEmpty()
}

func (m *SlotMachine) IsActive() bool {
	return m.syncQueue.IsActive()
}

func (m *SlotMachine) Stop() bool {
	m.logStop()
	return m.syncQueue.SetStopping()
}

func (m *SlotMachine) getScanAndMigrateCounts() (scanCount, migrateCount uint32) {
	v := atomic.LoadUint64(&m.scanAndMigrateCounts)
	return uint32(v), uint32(v >> 32)
}

func (m *SlotMachine) getScanCount() uint32 {
	v := atomic.LoadUint64(&m.scanAndMigrateCounts)
	return uint32(v)
}

func (m *SlotMachine) incScanCount() uint32 {
	for {
		v := atomic.LoadUint64(&m.scanAndMigrateCounts)
		vv := uint64(uint32(v)+1) | v&^math.MaxUint32
		if atomic.CompareAndSwapUint64(&m.scanAndMigrateCounts, v, vv) {
			return uint32(vv)
		}
	}
}

func (m *SlotMachine) incMigrateCount() uint32 {
	return uint32(atomic.AddUint64(&m.scanAndMigrateCounts, 1<<32) >> 32)
}

func (m *SlotMachine) CopyConfig() SlotMachineConfig {
	return m.config
}

/* ------- Methods for dependency injections - safe for concurrent use ------------- */

type dependencyKey string // is applied to avoid key interference with aliases

func (m *SlotMachine) FindDependency(id string) (interface{}, bool) {
	if v, ok := m.localRegistry.Load(dependencyKey(id)); ok {
		return v, true
	}
	if m.parentRegistry != nil {
		return m.parentRegistry.FindDependency(id)
	}
	return nil, false
}

func (m *SlotMachine) AddDependency(v interface{}) {
	injector.AddDependency(m, v)
}

func (m *SlotMachine) AddInterfaceDependency(v interface{}) {
	injector.AddInterfaceDependency(m, v)
}

func (m *SlotMachine) PutDependency(id string, v interface{}) {
	if id == "" {
		panic(throw.IllegalValue())
	}
	m.localRegistry.Store(dependencyKey(id), v)
}

func (m *SlotMachine) TryPutDependency(id string, v interface{}) bool {
	if id == "" {
		panic(throw.IllegalValue())
	}
	_, loaded := m.localRegistry.LoadOrStore(dependencyKey(id), v)
	return !loaded
}

func (m *SlotMachine) GetPublishedGlobalAliasAndBargeIn(key interface{}) (SlotLink, BargeInHolder) {
	av := m.getGlobalPublished(key)
	return av.Link, av.BargeIn
}

/* -- Methods to allocate slots ------------------------------ */

// SAFE for concurrent use
func (m *SlotMachine) allocateNextSlotID() SlotID {
	return m.config.SlotIDGenerateFn()
}

func (m *SlotMachine) _allocateNextSlotID() SlotID {
	for {
		r := atomic.LoadUint32((*uint32)(&m.lastSlotID))
		if r == math.MaxUint32 {
			panic("overflow")
		}
		if atomic.CompareAndSwapUint32((*uint32)(&m.lastSlotID), r, r+1) {
			return SlotID(r + 1)
		}
	}
}

// SAFE for concurrent use
func (m *SlotMachine) allocateSlot() *Slot {
	return m.slotPool.AllocateSlot(m, m.allocateNextSlotID())
}

/* -- Methods to dispose/reuse slots ------------------------------ */

func (m *SlotMachine) Cleanup(worker FixedSlotWorker) {
	m.slotPool.ScanAndCleanup(true, func(slot *Slot) {
		m.recycleSlot(slot, worker)
	}, func(slots []Slot) (isPageEmptyOrWeak, hasWeakSlots bool) {
		return m.verifyPage(slots, worker)
	})
	m.syncQueue.CleanupDetachQueues()
}

func (m *SlotMachine) verifyPage(slotPage []Slot, _ FixedSlotWorker) (isPageEmptyOrWeak, hasWeakSlots bool) {
	isPageEmptyOrWeak = true
	hasWeakSlots = false

	for i := range slotPage {
		slot := &slotPage[i]

		switch {
		case slot.isEmpty():
			continue
		case slot.isBusy():
			break
		case slot.step.Flags&StepWeak != 0:
			hasWeakSlots = true
			continue
		}
		return false, hasWeakSlots
	}
	return isPageEmptyOrWeak, hasWeakSlots
}

func (m *SlotMachine) stopPage(slotPage []Slot, w FixedSlotWorker) (isPageEmptyOrWeak, hasWeakSlots bool) {
	hasWorking := false

	for i := range slotPage {
		slot := &slotPage[i]

		switch isEmpty, isStarted, _ := slot._tryStartSlot(1); {
		case isEmpty:
			//continue
		case isStarted:
			m.recycleSlot(slot, w)
		default:
			hasWorking = true
		}
	}
	return !hasWorking, false
}

func (m *SlotMachine) recycleSlot(slot *Slot, worker FixedSlotWorker) {
	m.recycleSlotWithError(slot, worker, nil)
}

func (m *SlotMachine) recycleSlotWithError(slot *Slot, worker FixedSlotWorker, err error) {

	var link StepLink
	hasPanic := false
	func() {
		defer func() {
			recovered := recover()
			hasPanic = recovered != nil
			err = RecoverSlotPanicWithStack("internal panic - recycleSlot", recovered, err, InternalArea)
		}()

		link = slot.NewStepLink()
		slot.invalidateSlotID() // slotID is reset here and all links are invalid since this moment

		th := slot.defTerminate
		if th != nil {
			slot.defTerminate = nil // avoid self-loops
			m.runTerminationHandler(th, TerminationData{link, slot.parent, slot.ctx,
				slot.defResult, err})
		}

		if slot.hasAsyncOrBargeIn() || slot.slotFlags&(slotHadAliases) != 0 {
			defer m.syncQueue.FlushSlotDetachQueue(link.SlotLink)
		}

		if slot.slotFlags&slotHadAliases != 0 {
			// cleanup aliases associated with the slot
			// MUST happen before releasing of dependencies
			m.unregisterBoundAliases(link.SlotID())
		}

		{
			// cleanup synchronization dependency
			if slot.dependency != nil {
				released := slot._releaseAllDependency()
				m.activateDependants(released, link.SlotLink, worker)
			}
		}

		{ // cleanup queues
			if slot.isQueueHead() {
				s := slot.removeHeadedQueue()
				m._activateDependantChain(s, worker)
			} else {
				slot.removeFromQueue()
			}
		}
	}()

	if hasPanic {
		m.logCritical(link, "recycle", err)
	} else {
		slot.logInternal(link, "recycle", err)
	}
	m._recycleSlot(slot)
}

// SAFE for concurrent use
// This method can be called concurrently but ONLY to release new (empty) slots - slot MUST NOT have any kind of dependencies
func (m *SlotMachine) recycleEmptySlot(slot *Slot, err error) {
	if slot.slotFlags != 0 {
		panic("illegal state")
	}

	th := slot.defTerminate
	if th != nil { // it can be already set by construction - we must invoke it
		slot.defTerminate = nil // avoid self-loops
		m.runTerminationHandler(th, TerminationData{slot.NewStepLink(), slot.parent, slot.ctx,
			slot.defResult, err})
	}

	// slot.invalidateSlotID() // empty slot doesn't need early invalidation

	m._recycleSlot(slot) // SAFE for concurrent use
}

func (m *SlotMachine) _recycleSlot(slot *Slot) {
	slot.dispose()               // check state and cleanup fields
	m.slotPool.RecycleSlot(slot) // SAFE for concurrent use
}

func (m *SlotMachine) OccupiedSlotCount() int {
	return m.slotPool.Count()
}

func (m *SlotMachine) AllocatedSlotCount() int {
	return m.slotPool.Capacity()
}

/* -- General purpose synchronization ------------------------------ */

func (m *SlotMachine) ScheduleCall(fn MachineCallFunc, isSignal bool) bool {
	if fn == nil {
		panic("illegal value")
	}
	callFn := func(_ SlotLink, worker FixedSlotWorker) {
		mc := machineCallContext{m: m, w: worker}
		err := mc.executeCall(fn)
		if err != nil {
			m.logInternal(StepLink{}, "schedule call execution", err)
		}
	}
	if isSignal {
		return m.syncQueue.AddAsyncSignal(SlotLink{}, callFn)
	}
	return m.syncQueue.AddAsyncUpdate(SlotLink{}, callFn)
}

// SAFE for concurrent use
func (m *SlotMachine) runTerminationHandler(th internalTerminationHandlerFunc, td TerminationData) {
	m.syncQueue.AddAsyncCallback(td.Slot.SlotLink, func(_ SlotLink, _ DetachableSlotWorker) bool {
		err := func() (err error) {
			defer func() {
				err = RecoverSlotPanicWithStack("termination handler", recover(), nil, ErrorHandlerArea)
			}()
			th(td, FixedSlotWorker{})
			return nil
		}()
		if err != nil {
			m.logInternal(StepLink{SlotLink: td.Slot.SlotLink}, "failed termination handler", err)
		}
		return true
	})
}

/* -- Methods to create and start new machines ------------------------------ */

func (m *SlotMachine) AddNew(ctx context.Context, sm StateMachine, defValues CreateDefaultValues) (SlotLink, bool) {
	switch {
	case ctx != nil:
		defValues.Context = ctx
	case defValues.Context == nil:
		panic("illegal value")
	}

	link, ok := m.prepareNewSlotWithDefaults(nil, nil, sm, defValues)
	if ok {
		m.syncQueue.AddAsyncUpdate(link, m._startAddedSlot)
	}
	return link, ok
}

func (m *SlotMachine) AddNewByFunc(ctx context.Context, cf CreateFunc, defValues CreateDefaultValues) (SlotLink, bool) {
	switch {
	case ctx != nil:
		defValues.Context = ctx
	case defValues.Context == nil:
		panic("illegal value")
	}

	link, ok := m.prepareNewSlotWithDefaults(nil, cf, nil, defValues)
	if ok {
		m.syncQueue.AddAsyncUpdate(link, m._startAddedSlot)
	}
	return link, ok
}

func (m *SlotMachine) AddNested(_ AdapterID, parent SlotLink, cf CreateFunc) (SlotLink, bool) {
	if parent.IsZero() {
		panic("illegal value")
	}
	// TODO PLAT-25 pass adapterID into injections?

	link, ok := m.prepareNewSlot(nil, cf, nil, CreateDefaultValues{Parent: parent})
	if ok {
		m.syncQueue.AddAsyncUpdate(link, m._startAddedSlot)
	}
	return link, ok
}

func (m *SlotMachine) prepareNewSlotWithDefaults(creator *Slot, fn CreateFunc, sm StateMachine, defValues CreateDefaultValues) (SlotLink, bool) {
	return m.prepareNewSlot(creator, fn, sm, defValues)
}

// caller MUST be busy-holder of both creator and slot, then this method is SAFE for concurrent use
func (m *SlotMachine) prepareNewSlot(creator *Slot, fn CreateFunc, sm StateMachine, defValues CreateDefaultValues) (SlotLink, bool) {
	switch {
	case (fn == nil) == (sm == nil):
		panic(throw.IllegalValue())
	case !m.IsActive():
		return SlotLink{}, false
	}

	slot := m.allocateSlot()
	defer func() {
		if slot != nil {
			m.recycleEmptySlot(slot, nil) // all construction errors are reported to caller
		}
	}()

	slot.parent = defValues.Parent
	slot.ctx = defValues.Context

	// terminate handler must be executed even if construction has failed
	slot.defResult = defValues.TerminationResult
	if tfn := defValues.TerminationHandler; tfn != nil {
		slot.defTerminate = func(data TerminationData, _ FixedSlotWorker) {
			tfn(data)
		}
	}

	switch {
	case slot.ctx != nil:
		//
	case creator != nil:
		slot.ctx = creator.ctx
	case slot.parent.IsValid():
		// TODO PLAT-26 this can be racy, but under very awkward conditions:
		// 1. SM1 is from another SlotMachine running under a different worker
		// 2. SM1 has not finished its construction
		// 3. SM1 has provided its SlotLink to another party
		// 4. the other party has created a new SM2 with SM1 as parent, but didn't provide context.Context
		// 5. SM1 has called ConstructionContext.SetContext() while SM2 is being created
		slot.ctx = slot.parent.s.ctx
	}
	if slot.ctx == nil {
		slot.ctx = context.Background() // TODO PLAT-24 provide SlotMachine context?
	}

	link := slot.NewLink()
	if initFn := slot.prepareSlotInit(creator, fn, sm, defValues); initFn != nil {
		// final touch
		slot.step = SlotStep{Transition: initFn.defaultInit}
		slot.stepDecl = &defaultInitDecl
		slot = nil // protect from defer
		return link, true
	}
	return link, false // slot will be released by defer
}

func (m *SlotMachine) prepareStepLogger(slot *Slot, sm StateMachine, tracerID TracerID) {
	var stepLoggerFactory StepLoggerFactoryFunc
	if m.config.SlotMachineLogger != nil {
		stepLoggerFactory = m.config.SlotMachineLogger.CreateStepLogger
	}

	if stepLogger, ok := slot.declaration.GetStepLogger(slot.ctx, sm, tracerID, stepLoggerFactory); ok {
		slot.stepLogger = stepLogger
	} else if stepLoggerFactory != nil {
		slot.stepLogger = stepLoggerFactory(slot.ctx, sm, tracerID)
	}
	if slot.stepLogger == nil && len(tracerID) > 0 {
		slot.stepLogger = StepLoggerStub{tracerID}
	}
}

func (m *SlotMachine) prepareInjects(link SlotLink, sm StateMachine, mode DependencyInheritanceMode, isReplacement bool,
	creatorInheritable, constructorOverrides map[string]interface{},
) (map[string]interface{}, []interface{}) {

	var overrides []map[string]interface{}
	if len(constructorOverrides) > 0 {
		overrides = append(overrides, constructorOverrides)
	}
	if mode&InheritResolvedDependencies != 0 && len(creatorInheritable) > 0 {
		overrides = append(overrides, creatorInheritable)
	}

	localDeps := injector.NewMultiMapRegistry(overrides)

	var addedInjects []interface{}

	dResolver := injector.NewDependencyResolver(sm, m, localDeps, func(_ string, v interface{}, from injector.DependencyOrigin) {
		if from&(injector.DependencyFromLocal|injector.DependencyFromProvider) != 0 {
			addedInjects = append(addedInjects, v)
		}
	})
	dInjector := injector.NewDependencyInjectorFor(&dResolver)

	link.s.declaration.InjectDependencies(sm, link, dInjector)

	switch {
	case isReplacement:
		// replacing SM should take all inherited dependencies, even if unused
		if mode&copyAllDependencies != 0 {
			for _, o := range overrides {
				dResolver.ResolveAndMerge(o)
			}
		}
		if mode&DiscardResolvedDependencies != 0 {
			return nil, addedInjects
		}

	case mode&DiscardResolvedDependencies != 0:
		return nil, addedInjects

	case mode&copyAllDependencies != 0:
		localInjects := addedInjects // keep only injects that were explicitly used
		for _, o := range overrides {
			dResolver.ResolveAndMerge(o)
		}
		return dResolver.Flush(), localInjects
	}

	return dResolver.Flush(), addedInjects
}

func (m *SlotMachine) _boostNewSlot(slot *Slot) {
	if bp := m.boostPermitLatest; bp != nil && slot.boost == nil {
		bp.use()
		slot.boost = &bp.boostPermit
		if slot.boost.isActive() {
			slot.slotFlags |= slotIsBoosted
		}
	}
}

func (m *SlotMachine) startNewSlot(slot *Slot, worker FixedSlotWorker) {
	slot.ensureInitializing()
	m._boostNewSlot(slot)
	m.stopSlotWorking(slot, 0, worker)
	m.updateSlotQueue(slot, worker, activateSlot)
}

func (m *SlotMachine) startNewSlotByDetachable(slot *Slot, postInitFn PostInitFunc, w DetachableSlotWorker) {
	if postInitFn != nil {
		m._executeSlotInitByCreator(slot, postInitFn, w)
		return
	}

	slot.ensureInitializing()
	if !w.NonDetachableCall(func(worker FixedSlotWorker) {
		m._boostNewSlot(slot)
		m.stopSlotWorking(slot, 0, worker)
		m.updateSlotQueue(slot, worker, activateSlot)
	}) {
		m.syncQueue.AddAsyncUpdate(slot.NewLink(), m._startAddedSlot)
	}
}

func (m *SlotMachine) _startAddedSlot(link SlotLink, worker FixedSlotWorker) {
	if !link.IsValid() {
		panic("unexpected")
	}
	slot := link.s
	slot.ensureInitializing()
	m._boostNewSlot(slot)
	m.stopSlotWorking(slot, 0, worker)
	list := m._updateSlotQueue(slot, false, activateSlot)
	if list != nil {
		panic(throw.Impossible())
	}
}

/* -- Methods to manage processing queues, activate/deactivate slots ------------------------------ */

type slotActivationMode uint8

const (
	deactivateSlot slotActivationMode = iota
	activateSlot
	activateHotWaitSlot
)

func (m *SlotMachine) updateSlotQueue(slot *Slot, w FixedSlotWorker, activation slotActivationMode) {
	m.ensureLocal(slot)

	s := m._updateSlotQueue(slot, slot.isInQueue(), activation)
	m._activateDependantChain(s, w)
}

func (m *SlotMachine) _updateSlotQueue(slot *Slot, inplaceUpdate bool, activation slotActivationMode) *Slot {
	if !slot.isQueueHead() {
		if inplaceUpdate {
			switch activation {
			case activateSlot:
				switch slot.QueueType() {
				case ActiveSlots:
					m.hotWaitOnly = false
					return nil
				case WorkingSlots:
					return nil
				}
			case activateHotWaitSlot:
				if slot.QueueType() == ActiveSlots {
					return nil
				}
			}
			slot.removeFromQueue()
		} else {
			slot.ensureNotInQueue()
		}

		if activation == deactivateSlot {
			return nil
		}
		m._activateSlot(slot, activation)
		return nil
	}

	if slot.QueueType() != ActivationOfSlot {
		panic("illegal state")
	}

	if activation == deactivateSlot {
		if !inplaceUpdate {
			slot.ensureNotInQueue()
		}
		return nil
	}

	nextDep := slot.removeHeadedQueue()
	m._activateSlot(slot, activation)
	return nextDep
}

func (m *SlotMachine) _activateSlot(slot *Slot, mode slotActivationMode) {
	switch {
	case mode == activateHotWaitSlot:
		// hot wait ignores boosted to reduce interference
		switch {
		case slot.isPriority():
			m.prioritySlots.AddLast(slot)
		// case slot.isBoosted():
		//	m.boostedSlots.AddLast(slot)
		default:
			m.activeSlots.AddLast(slot)
		}
	case slot.isLastScan(m.getScanCount()):
		m.hotWaitOnly = false
		switch {
		case slot.isPriority():
			m.prioritySlots.AddLast(slot)
		case slot.isBoosted():
			m.boostedSlots.AddLast(slot)
		default:
			m.activeSlots.AddLast(slot)
		}
	default:
		// addSlotToWorkingQueue
		if slot.isPriority() {
			m.workingSlots.AddFirst(slot)
		} else {
			m.workingSlots.AddLast(slot)
		}
	}
}

/* ---- slot state updates and error handling ---------------------------- */

func (m *SlotMachine) applyStateUpdate(slot *Slot, stateUpdate StateUpdate, w FixedSlotWorker) bool {
	m.ensureLocal(slot)

	if stateUpdate.IsEmpty() {
		return true
	}

	var err error
	isAvailable := false
	isPanic := false

	func() {
		defer func() {
			recovered := recover()
			isPanic = recovered != nil
			err = RecoverSlotPanicWithStack("apply state update panic", recovered, err, InternalArea)
		}()
		isAvailable, err = typeOfStateUpdate(stateUpdate).Apply(slot, stateUpdate, w)
	}()

	if err == nil {
		return isAvailable
	}

	return m.handleSlotUpdateError(slot, w, stateUpdate, isPanic, err)
}

func (m *SlotMachine) handleSlotUpdateError(slot *Slot, worker FixedSlotWorker, stateUpdate StateUpdate, isPanic bool, err error) bool {
	canRecover := false
	area := StateArea
	var slotError SlotPanicError
	if errors.As(err, &slotError) && slotError.Area != 0 {
		area = slotError.Area
	}

	canRecover = area.CanRecoverByHandler()

	eh := slot.getErrorHandler()
	for repeated, ms := false, slot.stateStack; ; ms = ms.stateStack {
		action := ErrorHandlerDefault
		if eh != nil {
			fc := failureContext{isPanic: isPanic, area: area, canRecover: canRecover, err: err, result: slot.defResult}

			ok := false
			if ok, action, err = fc.executeFailure(eh); ok {
				// only change result on success of the handler
				slot.defResult = fc.result
			} else {
				action = ErrorHandlerDefault
			}
		}

		switch {
		case !repeated:
			switch action {
			case ErrorHandlerRecover, ErrorHandlerRecoverAndWakeUp:
				switch {
				case !canRecover:
					slot.logStepError(errorHandlerRecoverDenied, stateUpdate, area, err)
				case action == ErrorHandlerRecoverAndWakeUp:
					slot.activateSlot(worker)
					fallthrough
				default:
					slot.logStepError(ErrorHandlerRecover, stateUpdate, area, err)
					return true
				}
			case ErrorHandlerMute:
				slot.logStepError(ErrorHandlerMute, stateUpdate, area, err)
			default:
				slot.logStepError(ErrorHandlerDefault, stateUpdate, area, err)
			}

			if slot.hasSubroutine() && area.CanRecoverBySubroutine() {
				slot.prepareSubroutineExit(err)
				m.updateSlotQueue(slot, worker, activateSlot)
				return true
			}
		case action == ErrorHandlerRecover || action == ErrorHandlerRecoverAndWakeUp:
			slot.logStepError(errorHandlerRecoverDenied, stateUpdate, area, err)
		}
		if ms == nil {
			break
		}
		eh = ms.defErrorHandler
		repeated = true
	}

	m.recycleSlotWithError(slot, worker, err)
	return false
}

func (m *SlotMachine) logCritical(link StepLink, msg string, err error) {
	if sml := m.config.SlotMachineLogger; sml != nil {
		sml.LogMachineCritical(SlotMachineData{m.getScanCount(), link, err}, msg)
	}
}

func (m *SlotMachine) logInternal(link StepLink, msg string, err error) {
	if sml := m.config.SlotMachineLogger; sml != nil {
		sml.LogMachineInternal(SlotMachineData{m.getScanCount(), link, err}, msg)
	}
}

func (m *SlotMachine) logStop() {
	if sml := m.config.SlotMachineLogger; sml != nil {
		sml.LogStopping(m)
	}
}

/* ----- Time operations --------------------------- */

func minTime(t1, t2 time.Time) time.Time {
	if t1.IsZero() {
		return t2
	}
	if t2.IsZero() || t1.Before(t2) {
		return t1
	}
	return t2
}

func (m *SlotMachine) toRelativeTime(t time.Time) uint32 {
	if m.scanStartedAt.IsZero() {
		panic("illegal state")
	}
	if t.IsZero() {
		return 0
	}

	d := t.Sub(m.scanStartedAt)
	if d <= time.Microsecond {
		return 1
	}
	d = 1 + d/time.Microsecond
	if d > math.MaxUint32 {
		panic("illegal value")
	}
	return uint32(d)
}

func (m *SlotMachine) fromRelativeTime(rel uint32) time.Time {
	switch rel {
	case 0:
		return time.Time{}
	case 1:
		return m.scanStartedAt
	}
	return m.scanStartedAt.Add(time.Duration(rel-1) * time.Microsecond)
}

/* ---- Unsorted ---------------------------- */

// nolint:unused
func (m *SlotMachine) wakeupOnDeactivationOf(slot *Slot, waitOn SlotLink, worker FixedSlotWorker) {
	if waitOn.s == slot || !waitOn.IsValid() {
		// don't wait for self
		// don't wait for an expired slot
		m.updateSlotQueue(slot, worker, activateSlot)
		return
	}

	if waitOnM := waitOn.getActiveMachine(); waitOnM != nil {
		wakeupLink := slot.NewLink()
		waitOnM._wakeupOnDeactivateAsync(wakeupLink, waitOn)
	}
}

// waitOn MUST belong to this machine!
// nolint:unused
func (m *SlotMachine) _wakeupOnDeactivateAsync(wakeUp, waitOn SlotLink) {
	m.syncQueue.AddAsyncCallback(waitOn, func(waitOn SlotLink, worker DetachableSlotWorker) bool {
		switch {
		case !wakeUp.IsValid():
			// requester is dead - no need to to anything
			return true
		case !worker.IsZero() && waitOn.isValidAndBusy():
			// have to wait further, add this back
			return false
		}

		wakeUpM := wakeUp.getActiveMachine()
		switch {
		case wakeUpM == nil:
			return true
		case worker.IsZero():
			break
		case waitOn.isMachine(wakeUpM):
			if worker.NonDetachableCall(wakeUp.activateSlot) {
				return true
			}
		default:
			if worker.NonDetachableOuterCall(wakeUpM, wakeUp.activateSlot) {
				return true
			}
		}
		wakeUpM.syncQueue.AddAsyncUpdate(wakeUp, SlotLink.activateSlot)
		return true
	})
}

func (m *SlotMachine) useSlotAsShared(link SharedDataLink, accessFn SharedDataFunc, worker DetachableSlotWorker) SharedAccessReport {
	isValid, isBusy := link.link.getIsValidAndBusy()
	if !isValid {
		return SharedSlotAbsent
	}
	tm, data := link.getDataAndMachine()
	switch {
	case data == nil:
		return SharedSlotAbsent
	case tm == nil:
		return SharedSlotAbsent
	case isBusy:
		if m == tm {
			return SharedSlotLocalBusy
		}
		return SharedSlotRemoteBusy
	case m == tm:
		if m._useSlotAsShared(link.link, link.flags, data, accessFn, worker) {
			return SharedSlotLocalAvailable
		}
		return SharedSlotLocalBusy
	}

	ok := false
	// first we try to access gracefully the slot of another machine
	switch wasExecuted, wasDetached := worker.DetachableOuterCall(tm, func(tw DetachableSlotWorker) {
		ok = tm._useSlotAsShared(link.link, link.flags, data, accessFn, tw)
	}); {
	case wasExecuted:
		//
	case wasDetached:
		return SharedSlotRemoteBusy
	default:
		// as worker can't help us, then we do it in a hard way
		// this may be inefficient under high parallelism, but this isn't our case
		ok = tm._useSlotAsShared(link.link, link.flags, data, accessFn, DetachableSlotWorker{})
	}
	if ok {
		return SharedSlotRemoteAvailable
	}
	return SharedSlotRemoteBusy
}

func (m *SlotMachine) _useSlotAsShared(link SlotLink, flags ShareDataFlags, data interface{}, accessFn SharedDataFunc, worker DetachableSlotWorker) bool {
	slot, isStarted, _ := link.tryStartWorking()
	if !isStarted {
		return false
	}

	defer slot.stopWorking()
	wakeUp := accessFn(data)

	if !worker.IsZero() {
		m.syncQueue.ProcessSlotCallbacksByDetachable(link, worker)
	}

	if !wakeUp && flags&ShareDataWakesUpAfterUse == 0 || slot.slotFlags&slotWokenUp != 0 {
		return true
	}
	slot.slotFlags |= slotWokenUp

	if !worker.IsZero() && worker.NonDetachableCall(slot.activateSlot) {
		return true
	}

	stepLink := slot.NewStepLink() // remember the current step to avoid "back-fire" activation
	m.syncQueue.AddAsyncUpdate(stepLink.SlotLink, stepLink.activateSlotStepWithSlotLink)
	return true
}

// MUST match SlotMachine._executeSlot for short-looping
func (m *SlotMachine) stopSlotWorking(slot *Slot, prevStepNo uint32, worker FixedSlotWorker) {

	if !slot.needsReleaseOnStepping(prevStepNo) {
		slot.stopWorking()
		return
	}

	released := slot._releaseAllDependency()
	link := slot.NewLink()

	// MUST NOT access slot fields beyond this point
	slot.stopWorking()

	m.activateDependants(released, link, worker)
}

func (s *Slot) needsReleaseOnStepping(prevStepNo uint32) bool {
	switch dep := s.dependency; {
	case dep == nil:
		return false
	case !dep.IsReleaseOnStepping():
		return false
	}

	switch _, newStepNo, _ := s._getState(); {
	case newStepNo == prevStepNo:
		// step wasn't changed
		return false
	case prevStepNo == 0 && newStepNo <= 1:
		// step is initialization (which is considered as a part of creation)
		return false
	}
	return true
}

func (m *SlotMachine) _activateDependantChain(chain *Slot, worker FixedSlotWorker) {
	for chain != nil {
		s := chain
		// we MUST cut the slot out of chain before any actions on the slot
		chain = chain._cutNext()

		switch slotM := s.getMachine(); {
		case m == slotM:
			s.activateSlot(worker)
		case slotM == nil:
			// the other machine is has stopped ungracefully?
		case worker.OuterCall(slotM, s.activateSlot):
			//
		default:
			link := s.NewStepLink() // remember the current step to avoid "back-fire" activation
			slotM.syncQueue.AddAsyncUpdate(link.SlotLink, link.activateSlotStepWithSlotLink)
		}
	}
}

func (m *SlotMachine) activateDependants(links []StepLink, ignore SlotLink, worker FixedSlotWorker) {
	for _, link := range links {
		switch {
		case link.SlotLink == ignore:
			continue
		case link.isMachine(m):
			// slot will be activated if it is at the same step as it was when we've decided to activate it
			link.activateSlotStep(worker)
		default:
			switch linkM := link.getActiveMachine(); {
			case linkM == nil:
				//
			case worker.OuterCall(linkM, link.activateSlotStep):
				//
			default:
				linkM.syncQueue.AddAsyncUpdate(link.SlotLink, link.activateSlotStepWithSlotLink)
			}
		}
	}
}

func (m *SlotMachine) activateDependantByDetachable(links []StepLink, ignore SlotLink, worker DetachableSlotWorker) bool {
	if len(links) == 0 {
		return false
	}

	if worker.NonDetachableCall(func(worker FixedSlotWorker) {
		m.activateDependants(links, ignore, worker)
	}) {
		return true
	}

	if !m.syncQueue.AddAsyncUpdate(SlotLink{}, func(_ SlotLink, worker FixedSlotWorker) {
		m.activateDependants(links, ignore, worker)
	}) {
		// don't loose activations of other SlotMachines because of stopping of this one
		activateDependantWithoutWorker(links, ignore)
	}
	return true
}

func activateDependantWithoutWorker(links []StepLink, ignore SlotLink) {
	n := len(links)
	lm := links[0].getActiveMachine()
	from := 0
	for i := 1; i <= n; i++ {
		var m *SlotMachine
		if i < n {
			m = links[i].getActiveMachine()
		}
		switch {
		case m == lm:
			continue
		case lm != nil:
			subset := links[from:i]
			lm.syncQueue.AddAsyncUpdate(SlotLink{}, func(_ SlotLink, worker FixedSlotWorker) {
				lm.activateDependants(subset, ignore, worker)
			})
		}
		lm = m
		from = i
	}
}


func (m *SlotMachine) GetStoppingSignal() <-chan struct{} {
	return m.syncQueue.GetStoppingSignal()
}

// Must support nil receiver
func (m *SlotMachine) GetMachineID() string {
	return fmt.Sprintf("%p", m)
}

func (m *SlotMachine) getAdapterLogging() bool {
	return m.config.LogAdapterCalls
}

// UNSAFE!
func (m *SlotMachine) HasPriorityWork() bool {
	return !m.prioritySlots.IsEmpty() || !m.workingSlots.IsEmpty()
}

func (m *SlotMachine) AsHolder() SlotMachineHolder {
	return slotMachineHolder{m}
}
