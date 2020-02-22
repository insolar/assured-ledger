// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import "context"

type bargeInBuilder struct {
	c      slotContext
	parent *slotContext
	link   StepLink
}

func (b bargeInBuilder) with(stateUpdate StateUpdate) BargeInFunc {
	b.parent.ensureValid()
	defer b.c.setDiscarded()
	return b.c.s.machine.createLightBargeIn(b.link,
		b.c.ensureAndPrepare(b.c.s, stateUpdate))
}

func (b bargeInBuilder) WithError(err error) BargeInFunc {
	return b.with(b.c.Error(err))
}

func (b bargeInBuilder) WithStop() BargeInFunc {
	return b.with(b.c.Stop())
}

func (b bargeInBuilder) WithWakeUp() BargeInFunc {
	return b.with(b.c.WakeUp())
}

func (b bargeInBuilder) WithJumpExt(step SlotStep) BargeInFunc {
	return b.with(b.c.JumpExt(step))
}

func (b bargeInBuilder) WithJump(fn StateFunc) BargeInFunc {
	return b.with(b.c.Jump(fn))
}

/* ========================================================================= */

var _ BargeInContext = &bargingInContext{}

type bargingInContext struct {
	slotContext
	param      interface{}
	atOriginal bool
}

func (p *bargingInContext) EventParam() interface{} {
	p.ensureAtLeast(updCtxBargeIn)
	return p.param
}

func (p *bargingInContext) IsAtOriginalStep() bool {
	p.ensure(updCtxBargeIn)
	return p.atOriginal
}

func (p *bargingInContext) Log() Logger {
	p.ensureAtLeast(updCtxBargeIn)
	return p._newLogger()
}

func (p *bargingInContext) executeBargeIn(fn BargeInApplyFunc) (stateUpdate StateUpdate) {
	p.setMode(updCtxBargeIn)
	defer func() {
		p.discardAndUpdate("barge in", recover(), &stateUpdate, BargeInArea)
	}()

	return p.ensureAndPrepare(p.s, fn(p))
}

func (p *bargingInContext) executeBargeInNow(fn BargeInApplyFunc) (stateUpdate StateUpdate) {
	p.setMode(updCtxBargeIn)
	defer p.setDiscarded()

	return p.ensureAndPrepare(p.s, fn(p))
}

/* ========================================================================= */

type subroutineExitContext struct {
	bargingInContext
	err error
}

func (p *subroutineExitContext) GetError() error {
	p.ensureAtLeast(updCtxSubrExit)
	return p.err
}

func (p *subroutineExitContext) executeSubroutineExit(fn SubroutineExitFunc) (stateUpdate StateUpdate) {
	p.setMode(updCtxSubrExit)
	defer func() {
		p.discardAndUpdate("subroutine exit", recover(), &stateUpdate, StateArea)
	}()

	return p.ensureAndPrepare(p.s, fn(p))
}

/* ========================================================================= */

var _ AsyncResultContext = &asyncResultContext{}

type asyncResultContext struct {
	contextTemplate
	s      *Slot
	wakeup bool
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
