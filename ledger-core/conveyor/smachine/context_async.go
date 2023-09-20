package smachine

import "context"

type bargeInBuilder struct {
	parent *slotContext
	link   StepLink
}

func (b bargeInBuilder) with(stateUpdate StateUpdate) BargeIn {
	b.parent.ensureValid()

	su := b.parent.ensureAndPrepare(b.parent.s, stateUpdate)
	return b.parent.s.machine.createLightBargeIn(b.link, su)
}

func (b bargeInBuilder) WithError(err error) BargeIn {
	return b.with(b.parent.Error(err))
}

func (b bargeInBuilder) WithStop() BargeIn {
	return b.with(b.parent.Stop())
}

func (b bargeInBuilder) WithWakeUp() BargeIn {
	return b.with(b.parent.WakeUp())
}

func (b bargeInBuilder) WithJumpExt(step SlotStep) BargeIn {
	return b.with(b.parent.JumpExt(step))
}

func (b bargeInBuilder) WithJump(fn StateFunc) BargeIn {
	return b.with(b.parent.Jump(fn))
}

/* ========================================================================= */

var _ BargeInContext = &bargingInContext{}

type bargingInContext struct {
	slotContext
	atOriginal bool
	callerSM bool
}

func (p *bargingInContext) AffectedStep() SlotStep {
	p.ensure(updCtxBargeIn)
	return p.affectedStep(p.callerSM)
}

func (p *bargingInContext) IsAtOriginalStep() bool {
	p.ensure(updCtxBargeIn)
	return p.atOriginal
}

func (p *bargingInContext) Log() Logger {
	p.ensureAtLeast(updCtxBargeIn)
	return p._newLogger()
}

func (p *bargingInContext) executeBargeIn(fn BargeInCallbackFunc) (stateUpdate StateUpdate) {
	p.setMode(updCtxBargeIn)
	defer func() {
		stateUpdate = p.discardAndUpdate("barge in", recover(), stateUpdate, BargeInArea)
	}()

	return p.ensureAndPrepare(p.s, fn(p))
}

func (p *bargingInContext) executeBargeInDirect(fn BargeInCallbackFunc) (stateUpdate StateUpdate) {
	p.setMode(updCtxBargeIn)
	defer p.setDiscarded()

	return p.ensureAndPrepare(p.s, fn(p))
}

/* ========================================================================= */

var _ SubroutineStartContext = &subroutineStartContext{}

type subroutineStartContext struct {
	slotContext
	cleanupMode SubroutineCleanupMode
}

func (p *subroutineStartContext) SetSubroutineCleanupMode(mode SubroutineCleanupMode) {
	p.ensure(updCtxSubrStart)
	p.cleanupMode = mode
}

func (p *subroutineStartContext) executeSubroutineStart(fn SubroutineStartFunc) InitFunc {
	p.setMode(updCtxSubrStart)
	defer func() {
		p.setDiscarded()
	}()
	return fn(p)
}

/* ========================================================================= */

type subroutineExitContext struct {
	bargingInContext
	param interface{}
	err   error
}

func (p *subroutineExitContext) EventParam() interface{} {
	p.ensureAtLeast(updCtxSubrExit)
	return p.param
}

func (p *subroutineExitContext) GetError() error {
	p.ensureAtLeast(updCtxSubrExit)
	return p.err
}

func (p *subroutineExitContext) executeSubroutineExit(fn SubroutineExitFunc) (stateUpdate StateUpdate) {
	p.setMode(updCtxSubrExit)
	defer func() {
		stateUpdate = p.discardAndUpdate("subroutine exit", recover(), stateUpdate, StepArea)
	}()

	return p.ensureAndPrepare(p.s, fn(p))
}

/* ========================================================================= */

var _ AsyncResultContext = &asyncResultContext{}

type asyncResultContext struct {
	contextTemplate
	s      *Slot
	wakeup bool
	w      DetachableSlotWorker
}

func (p *asyncResultContext) SlotLink() SlotLink {
	p.ensure(updCtxAsyncCallback)
	return p.s.NewLink()
}

func (p *asyncResultContext) ParentLink() SlotLink {
	p.ensure(updCtxAsyncCallback)
	return p.s.parent
}

func (p *asyncResultContext) GetContext() context.Context {
	p.ensure(updCtxAsyncCallback)
	return p.s.ctx
}

func (p *asyncResultContext) Log() Logger {
	p.ensure(updCtxAsyncCallback)
	return Logger{p.s.ctx, p}
}

func (p *asyncResultContext) getStepLogger() (StepLogger, StepLogLevel, uint32) {
	p.ensureAtLeast(updCtxAsyncCallback)
	return p.s.stepLogger, p.s.getStepLogLevel(), 0
}

func (p *asyncResultContext) getStepLoggerData() StepLoggerData {
	return p.s.newStepLoggerData(StepLoggerTrace, p.s.NewStepLink())
}

func (p *asyncResultContext) WakeUp() {
	p.ensure(updCtxAsyncCallback)
	p.wakeup = true
}

func (p *asyncResultContext) executeResult(fn AsyncResultFunc) bool {
	p.setMode(updCtxAsyncCallback)
	defer p.setDiscarded()

	fn(p)
	return p.wakeup
}

func (p *asyncResultContext) ReleaseAll() bool {
	p.ensure(updCtxAsyncCallback)
	return p.s.contextReleaseAll(p.w)
}

func (p *asyncResultContext) ApplyAdjustment(adj SyncAdjustment) bool {
	p.ensure(updCtxAsyncCallback)
	return p.s.contextApplyAdjustment(adj, p.w)
}
