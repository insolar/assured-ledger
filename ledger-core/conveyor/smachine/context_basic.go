// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"context"
	"math"
	"unsafe"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type contextTemplate struct {
	mode updCtxMode
}

// protects from taking a copy of a context
func (p *contextTemplate) getMarker() ContextMarker {
	return ContextMarker(unsafe.Pointer(p))
	// ContextMarker(unsafe.Pointer(p)) ^ atomicCounter.AddUint32(1) << 16
}

func (p *contextTemplate) ensureAndPrepare(s *Slot, stateUpdate StateUpdate) StateUpdate {
	stateUpdate.ensureMarker(p.getMarker())

	sut := typeOfStateUpdateForPrepare(p.mode, stateUpdate)
	sut.Prepare(s, &stateUpdate)

	return stateUpdate
}

func (p *contextTemplate) setMode(mode updCtxMode) {
	if mode == updCtxInactive {
		panic(throw.IllegalValue())
	}
	if p.mode != updCtxInactive {
		panic(throw.IllegalState())
	}
	p.mode = mode
}

func (p *contextTemplate) ensureAtLeast(mode updCtxMode) {
	if p.mode < mode {
		panic(throw.IllegalState())
	}
}

func (p *contextTemplate) ensure(mode0 updCtxMode) {
	if p.mode != mode0 {
		panic(throw.IllegalState())
	}
}

func (p *contextTemplate) ensureAny2(mode0, mode1 updCtxMode) {
	if p.mode != mode0 && p.mode != mode1 {
		panic(throw.IllegalState())
	}
}

func (p *contextTemplate) ensureAny3(mode0, mode1, mode2 updCtxMode) {
	if p.mode != mode0 && p.mode != mode1 && p.mode != mode2 {
		panic(throw.IllegalState())
	}
}

func (p *contextTemplate) ensureValid() {
	if p.mode <= updCtxDiscarded {
		panic(throw.IllegalState())
	}
}

func (p *contextTemplate) template(updType stateUpdKind) StateUpdateTemplate {
	return newStateUpdateTemplate(p.mode, p.getMarker(), updType)
}

func (p *contextTemplate) setDiscarded() {
	p.mode = updCtxDiscarded
}

func (p *contextTemplate) discardAndCapture(msg string, recovered interface{}, err *error, area SlotPanicArea) {
	p.mode = updCtxDiscarded
	if recovered == nil {
		return
	}
	*err = RecoverSlotPanic(msg, recovered, *err, area)
}

func (p *contextTemplate) discardAndUpdate(msg string, recovered interface{}, update StateUpdate, area SlotPanicArea) StateUpdate {
	p.mode = updCtxDiscarded
	return recoverSlotPanicAsUpdate(update, msg, recovered, nil, area)
}

/* ========================================================================= */

type slotContext struct {
	contextTemplate
	s *Slot
	w DetachableSlotWorker
}

func (p *slotContext) clone(mode updCtxMode) slotContext {
	p.ensureValid()
	return slotContext{s: p.s, w: p.w, contextTemplate: contextTemplate{mode: mode}}
}

func (p *slotContext) SlotLink() SlotLink {
	p.ensureValid()
	return p.s.NewLink()
}

func (p *slotContext) StepLink() StepLink {
	p.ensure(updCtxExec)
	return p.s.NewStepLink()
}

func (p *slotContext) GetContext() context.Context {
	p.ensureValid()
	return p.s.ctx
}

func (p *slotContext) ParentLink() SlotLink {
	p.ensureValid()
	return p.s.parent
}

func (p *slotContext) SetDefaultErrorHandler(fn ErrorHandlerFunc) {
	p.ensureAtLeast(updCtxInit)
	p.s.defErrorHandler = fn
}

func (p *slotContext) SetDefaultMigration(fn MigrateFunc) {
	p.ensureAtLeast(updCtxInit)
	p.s.defMigrate = fn
}

func (p *slotContext) SetDefaultFlags(flags StepFlags) {
	p.ensureAtLeast(updCtxInit)
	if flags&StepResetAllFlags != 0 {
		p.s.defFlags = flags &^ StepResetAllFlags
	} else {
		p.s.defFlags |= flags
	}
}

func (p *slotContext) SetDefaultTerminationResult(v interface{}) {
	p.ensureAtLeast(updCtxInit)
	p.s.defResult = v
}

func (p *slotContext) GetDefaultTerminationResult() interface{} {
	p.ensureAtLeast(updCtxInit)
	return p.s.defResult
}

func (p *slotContext) OverrideDynamicBoost(boosted bool) {
	p.ensureAtLeast(updCtxInit)
	if boosted {
		p.s.boost = activeBoost
	} else {
		p.s.boost = inactiveBoost
	}
}

func (p *slotContext) JumpExt(step SlotStep) StateUpdate {
	return p.template(stateUpdNext).newStep(step, nil)
}

func (p *slotContext) RestoreStep(step SlotStep) StateUpdate {
	return p.template(stateUpdRestore).newStepOnly(step)
}

func (p *slotContext) Jump(fn StateFunc) StateUpdate {
	return p.template(stateUpdNext).newStep(SlotStep{Transition: fn}, nil)
}

func (p *slotContext) Stop() StateUpdate {
	return p.template(stateUpdStop).newNoArg()
}

func (p *slotContext) Error(err error) StateUpdate {
	return p.template(stateUpdError).newError(err)
}

func (p *slotContext) Repeat(limit int) StateUpdate {
	ulimit := uint32(0)
	switch {
	case limit <= 0:
	case limit > math.MaxUint32:
		ulimit = math.MaxUint32
	default:
		ulimit = uint32(limit)
	}

	return p.template(stateUpdRepeat).newUint(ulimit)
}

func (p *slotContext) Stay() StateUpdate {
	return p.template(stateUpdNoChange).newNoArg()
}

func (p *slotContext) WakeUp() StateUpdate {
	return p.template(stateUpdWakeup).newNoArg()
}

func (p *slotContext) AffectedStep() SlotStep {
	p.ensureAny3(updCtxMigrate, updCtxBargeIn, updCtxFail)
	r := p.s.step
	r.Flags |= StepResetAllFlags

	switch {
	case p.mode == updCtxFail:
		// Slot will always wakeup on errors
	case p.s.QueueType() == NoQueue:
		r.Flags |= stepSleepState
	}

	return r
}

func (p *slotContext) NewChild(fn CreateFunc) SlotLink {
	return p._newChild(fn, nil, CreateDefaultValues{Context: p.s.ctx, Parent: p.s.NewLink()})
}

func (p *slotContext) NewChildExt(fn CreateFunc, defValues CreateDefaultValues) SlotLink {
	return p._newChild(fn, nil, defValues)
}

func (p *slotContext) InitChild(fn CreateFunc) SlotLink {
	return p.InitChildWithPostInit(fn, func() {})
}

func (p *slotContext) InitChildWithPostInit(fn CreateFunc, postInitFn PostInitFunc) SlotLink {
	if postInitFn == nil {
		panic(throw.IllegalValue())
	}
	return p._newChild(fn, postInitFn, CreateDefaultValues{Context: p.s.ctx, Parent: p.s.NewLink()})
}

func (p *slotContext) InitChildExt(fn CreateFunc, defValues CreateDefaultValues, postInitFn PostInitFunc) SlotLink {
	if postInitFn == nil {
		postInitFn = func() {}
	}
	return p._newChild(fn, postInitFn, defValues)
}

func (p *slotContext) _newChild(fn CreateFunc, postInitFn PostInitFunc, defValues CreateDefaultValues) SlotLink {
	p.ensureAny2(updCtxExec, updCtxFail)
	if fn == nil {
		panic("illegal value")
	}
	if len(defValues.TracerID) == 0 {
		defValues.TracerID = p.s.getTracerID()
	}

	m := p.s.machine
	link, ok := m.prepareNewSlotWithDefaults(p.s, fn, nil, defValues)
	if ok {
		m.startNewSlotByDetachable(link.s, postInitFn, p.w)
	}
	return link
}

func (p *slotContext) Log() Logger {
	p.ensureAtLeast(updCtxSubrStart)
	return p._newLogger()
}

func (p *slotContext) LogAsync() Logger {
	p.ensure(updCtxExec)
	logger, _ := p._newLoggerAsync()
	return logger
}

func (p *slotContext) _getLoggerCtx() context.Context {
	if p.s.stepLogger != nil {
		if ctx := p.s.stepLogger.GetLoggerContext(); ctx != nil {
			return ctx
		}
	}
	return p.s.ctx
}

func (p *slotContext) _newLogger() Logger {
	return Logger{p._getLoggerCtx(), p}
}

func (p *slotContext) _newLoggerAsync() (Logger, uint32) {
	logger := Logger{p.s.ctx, nil}
	stepLogger, level, _ := p.getStepLogger()
	if stepLogger == nil {
		return logger, 0
	}
	if ctx := stepLogger.GetLoggerContext(); ctx != nil {
		logger.ctx = ctx
	}

	fsl := fixedSlotLogger{logger: stepLogger, level: level, data: p.getStepLoggerData()}
	logger.ctx, fsl.logger = stepLogger.CreateAsyncLogger(logger.ctx, &fsl.data)
	if fsl.logger == nil || logger.ctx == nil {
		panic("illegal state - logger doesnt support async")
	}
	fsl.data.Flags |= StepLoggerDetached

	logger.logger = fsl
	return logger, fsl.data.StepNo.step
}

func (p *slotContext) getStepLogger() (StepLogger, StepLogLevel, uint32) {
	return p.s.stepLogger, p.s.getStepLogLevel(), 0
}

func (p *slotContext) getStepLoggerData() StepLoggerData {
	return p.s.newStepLoggerData(StepLoggerTrace, p.s.NewStepLink())
}

func (p *slotContext) SetLogTracing(b bool) {
	p.ensureAtLeast(updCtxInit)
	p.s.setTracing(b)
}

func (p *slotContext) UpdateDefaultStepLogger(updateFn StepLoggerUpdateFunc) {
	p.ensureAtLeast(updCtxInit)
	p.s.setStepLoggerAfterInit(updateFn)
}

func (p *slotContext) NewBargeInWithParam(applyFn BargeInApplyFunc) BargeInWithParam {
	p.ensureAtLeast(updCtxInit)
	return p.s.machine.createBargeIn(p.s.NewStepLink().AnyStep(), applyFn)
}

func (p *slotContext) NewBargeIn() BargeInBuilder {
	p.ensureAtLeast(updCtxInit)
	return &bargeInBuilder{p, p.s.NewStepLink().AnyStep()}
}

func (p *slotContext) NewBargeInThisStepOnly() BargeInBuilder {
	p.ensureAtLeast(updCtxExec)
	return &bargeInBuilder{p, p.s.NewStepLink()}
}

func (p *slotContext) CallBargeInWithParam(b BargeInWithParam, param interface{}) bool {
	p.ensureAny2(updCtxInit, updCtxExec)
	return b.callInline(p.s.machine, param, p.w)
}

func (p *slotContext) CallBargeIn(b BargeIn) bool {
	p.ensureAny2(updCtxInit, updCtxExec)
	return b.callInline(p.s.machine, p.w)
}

func (p *slotContext) CallSubroutine(ssm SubroutineStateMachine, migrateFn MigrateFunc, exitFn SubroutineExitFunc) StateUpdate {
	p.ensureAny2(updCtxExec, updCtxMigrate)
	nextStep := p.s.prepareSubroutineStart(ssm, exitFn, migrateFn)
	return p.template(stateUpdSubroutineStart).newStepOnly(nextStep)
}

func (p *slotContext) Check(link SyncLink) BoolDecision {
	p.ensureAtLeast(updCtxInit)

	if link.controller == nil {
		panic("illegal value")
	}

	dep := p.s.dependency
	if dep != nil {
		if d, ok := link.controller.UseDependency(dep, SyncIgnoreFlags).AsValid(); ok {
			return d
		}
	}
	return link.controller.CheckState()
}

func (p *slotContext) AcquireExt(link SyncLink, flags AcquireFlags) BoolDecision {
	sdf := SlotDependencyFlags(0)
	if flags&AcquireForThisStep != 0 {
		sdf |= SyncForOneStep
	}
	if flags&BoostedPriorityAcquire != 0 {
		sdf |= SyncPriorityBoosted
	}
	if flags&HighPriorityAcquire != 0 {
		sdf |= SyncPriorityHigh
	}
	return p.acquire(link, flags&AcquireAndRelease != 0, flags&NoPriorityAcquire != 0, sdf)
}


func (p *slotContext) Acquire(link SyncLink) BoolDecision {
	return p.acquire(link, false, false, 0)
}

func (p *slotContext) AcquireForThisStep(link SyncLink) BoolDecision {
	return p.acquire(link, false, false, SyncForOneStep)
}

func (p *slotContext) AcquireAndRelease(link SyncLink) BoolDecision {
	return p.acquire(link, true, false, 0)
}

func (p *slotContext) AcquireForThisStepAndRelease(link SyncLink) BoolDecision {
	return p.acquire(link, true, false, SyncForOneStep)
}

func (p *slotContext) acquire(link SyncLink, autoRelease, ignoreSlotPriority bool, flags SlotDependencyFlags) (d BoolDecision) {
	p.ensureAtLeast(updCtxInit)

	switch {
	case link.IsZero():
		panic(throw.IllegalValue())
	case ignoreSlotPriority:
	case p.s.isSyncPriority():
		flags |= SyncPriorityHigh
	case p.s.isSyncBoost():
		flags |= SyncPriorityBoosted
	}

	dep := p.s.dependency
	if dep == nil {
		d, p.s.dependency = link.controller.CreateDependency(p.s.NewLink(), flags)
		return d
	}
	if d, ok := link.controller.UseDependency(dep, flags).AsValid(); ok {
		return d
	}

	if !autoRelease {
		panic("SM has already acquired another sync or the same one but with incompatible flags")
	}

	slotLink := p.s.NewLink()
	p.s.dependency = nil
	d, p.s.dependency = link.controller.CreateDependency(slotLink, flags)

	postponed, released := dep.ReleaseAll()
	released = PostponedList(postponed).PostponedActivate(released)

	p.s.machine.activateDependantByDetachable(released, slotLink, p.w)

	return d
}

func (p *slotContext) Release(link SyncLink) bool {
	p.ensureAtLeast(updCtxInit)

	if link.IsZero() {
		panic("illegal value")
	}
	return p.release(link.controller)
}

func (p *slotContext) releaseAll() bool {
	if p.s.dependency == nil {
		return false
	}

	released := p.s._releaseAllDependency()
	p.s.machine.activateDependantByDetachable(released, p.s.NewLink(), p.w)
	return true
}

func (p *slotContext) release(controller DependencyController) bool {
	dep := p.s.dependency
	if dep == nil {
		return false
	}
	if !controller.UseDependency(dep, SyncIgnoreFlags).IsValid() {
		// mismatched sync object
		return false
	}

	released, wasReleased := p.s._releaseDependency(controller)
	p.s.machine.activateDependantByDetachable(released, p.s.NewLink(), p.w)
	return wasReleased
}

func (p *slotContext) ReleaseAll() bool {
	p.ensureValid()
	return p.releaseAll()
}

func (p *slotContext) ApplyAdjustment(adj SyncAdjustment) bool {
	p.ensureValid()

	if adj.controller == nil {
		panic(throw.IllegalValue())
	}

	released, activate := adj.controller.AdjustLimit(adj.adjustment, adj.isAbsolute)
	if activate {
		return p.s.machine.activateDependantByDetachable(released, p.s.NewLink(), p.w)
	}

	// actually, we MUST NOT stop a slot from outside
	return len(released) > 0
}

func ApplyAdjustmentAsync(adj SyncAdjustment) bool {
	if adj.controller == nil {
		panic(throw.IllegalValue())
	}

	released, activate := adj.controller.AdjustLimit(adj.adjustment, adj.isAbsolute)
	n := len(released)
	switch {
	case n == 0:
		return false
	case activate:
		// can only start slots
		activateDependantWithoutWorker(released, SlotLink{})
	}
	return true
}
