// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"math"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ ExecutionContext = &executionContext{}

type executionContext struct {
	slotContext
	countAsyncCalls uint16
	flags           LongRunFlags
}

func (p *executionContext) InitiateLongRun(flags LongRunFlags) {
	if p.flags != 0 {
		panic("illegal state - repeated call")
	}
	p.w.TryDetach(flags)
	p.flags = manualDetach | flags
}

func (p *executionContext) GetPendingCallCount() int {
	p.ensure(updCtxExec)
	return int(p.s.asyncCallCount) + int(p.countAsyncCalls)
}

func (p *executionContext) Jump(fn StateFunc) StateUpdate {
	// allow looping on jump
	return p.template(stateUpdNextLoop).newStepUint(SlotStep{Transition: fn}, math.MaxUint32)
}

func (p *executionContext) Yield() ConditionalBuilder {
	ncu := p.newConditionalUpdate(stateUpdNext)
	return &ncu
}

func (p *executionContext) Poll() ConditionalBuilder {
	ncu := p.newConditionalUpdate(stateUpdPoll)
	return &ncu
}

func (p *executionContext) Sleep() ConditionalBuilder {
	ncu := p.newConditionalUpdate(stateUpdSleep)
	return &ncu
}

func (p *executionContext) WaitAny() ConditionalBuilder {
	ncu := p.newConditionalUpdate(stateUpdWaitForEvent)
	return &ncu
}

func (p *executionContext) WaitAnyUntil(until time.Time) StateConditionalBuilder {
	ncu := p.newConditionalUpdate(stateUpdWaitForEvent)

	ncu.until = p.s.machine.toRelativeTime(until)
	if ncu.until != 0 && time.Until(until) <= 0 {
		ncu.decision = Passed
	}

	return &ncu
}

func (p *executionContext) newConditionalUpdate(updType stateUpdKind) conditionalUpdate {
	p.ensure(updCtxExec)
	return conditionalUpdate{template: newStateUpdateTemplate(p.mode, p.getMarker(), updType)}
}

func (p *executionContext) waitFor(link SlotLink, updMode stateUpdKind) StateConditionalBuilder {
	p.ensure(updCtxExec)
	if link.IsZero() {
		panic(throw.IllegalValue())
		//		return &conditionalUpdate{marker: p.getMarker()}
	}

	switch isValid, isBusy := link.getIsValidAndBusy(); {
	case !isValid:
		ncu := p.newConditionalUpdate(stateUpdNext)
		ncu.decision = Impossible
		return &ncu
	case !isBusy:
		ncu := p.newConditionalUpdate(stateUpdNext)
		ncu.decision = Passed
		return &ncu
	default:
		ncu := p.newConditionalUpdate(updMode)
		ncu.decision = NotPassed
		ncu.dependency = link
		return &ncu
	}
}

// EXPERIMENTAL! SM will apply an action chosen by the builder and wait for activation or stop of the given slot.
func (p *executionContext) WaitActivation(link SlotLink) StateConditionalBuilder {
	return p.waitFor(link, stateUpdWaitForActive)
}

func (p *executionContext) WaitShared(link SharedDataLink) StateConditionalBuilder {
	return p.waitFor(link.link, stateUpdWaitForIdle)
}

func (p *executionContext) UseShared(a SharedDataAccessor) SharedAccessReport {
	p.ensure(updCtxExec)

	switch a.accessByOwner(p.s) {
	case Passed:
		return SharedSlotAvailableAlways
	case Impossible:
		return SharedSlotAbsent
	}
	return p.s.machine.useSlotAsShared(a.link, a.accessFn, p.w)
}

func (p *executionContext) executeNextStep() (stateUpdate StateUpdate, sut StateUpdateType, asyncCallCount uint16) {
	p.setMode(updCtxExec)
	defer func() {
		stateUpdate = p.discardAndUpdate("execution", recover(), stateUpdate, StepArea)
	}()

	current := p.s.step

	stateUpdate = current.Transition(p).ensureMarker(p.getMarker())
	sut = typeOfStateUpdateForPrepare(p.mode, stateUpdate)
	sut.Prepare(p.s, &stateUpdate)

	return stateUpdate, sut, p.countAsyncCalls
}

/* ========================================================================= */

var _ ConditionalBuilder = &conditionalUpdate{}
var _ StateConditionalBuilder = &conditionalUpdate{}

type conditionalUpdate struct {
	template   StateUpdateTemplate
	kickOff    StepPrepareFunc
	dependency SlotLink
	until      uint32
	decision   Decision
}

func (c *conditionalUpdate) GetDecision() Decision {
	if c.decision.IsZero() {
		return NotPassed
	}
	return c.decision
}

func (c *conditionalUpdate) ThenRepeatOrJump(fn StateFunc) StateUpdate {
	if c.GetDecision().IsNotPassed() {
		return c.ThenRepeat()
	}
	return c.ThenJump(fn)
}

func (c *conditionalUpdate) ThenRepeatOrJumpExt(step SlotStep) StateUpdate {
	if c.GetDecision().IsNotPassed() {
		return c.ThenRepeat()
	}
	return c.ThenJumpExt(step)
}

func (c *conditionalUpdate) ThenJump(fn StateFunc) StateUpdate {
	return c.ThenJumpExt(SlotStep{Transition: fn})
}

func (c *conditionalUpdate) ThenJumpExt(step SlotStep) StateUpdate {
	step.ensureTransition()
	return c.then(step)
}

func (c *conditionalUpdate) ThenRepeat() StateUpdate {
	return c.then(SlotStep{})
}

func (c *conditionalUpdate) then(slotStep SlotStep) StateUpdate {
	if c.dependency.IsZero() {
		if c.until == 0 {
			return c.template.newStep(slotStep, c.kickOff)
		}
		return c.template.newStepUntil(slotStep, c.kickOff, c.until)
	}
	if c.until != 0 {
		panic(throw.IllegalValue())
	}
	return c.template.newStepLink(slotStep, c.dependency)
}
