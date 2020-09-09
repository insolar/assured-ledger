// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/conveyor/smachine.BargeInHolder -o ./ -s _mock.go -g
type BargeInHolder interface {
	StepLink() StepLink
	IsValid() bool
	CallWithParam(interface{}) bool
}

type BargeInNoArgHolder interface {
	StepLink() StepLink
	IsValid() bool
	Call() bool
}

func NewNoopBargeIn(link StepLink) BargeIn {
	if link.IsZero() {
		return BargeIn{}
	}

	return BargeIn{update: StateUpdate{
		marker: ContextMarker(link.step),
		link:   link.s,
		param0: uint32(link.id),
	}}
}

func NewNoopBargeInWithParam(link StepLink) BargeInWithParam {
	if link.IsZero() {
		return BargeInWithParam{}
	}

	return BargeInWithParam{link: link, applyFn: func(interface{}) BargeInCallbackFunc {
		return nil
	}}
}

type BargeIn struct {
	update StateUpdate
	marker subroutineMarker
}

func (v BargeIn) StepLink() StepLink {
	return StepLink{SlotLink: v.update.getLink(), step: uint32(v.update.marker)}
}

func (v BargeIn) getStateUpdate() (stateUpdate StateUpdate) {
	stateUpdate = v.update
	stateUpdate.link = nil
	stateUpdate.param0 = 0
	stateUpdate.marker = 0
	return stateUpdate
}

func (v BargeIn) IsZero() bool {
	return v.update.link == nil
}

func (v BargeIn) IsValid() bool {
	return v.StepLink().IsAtStep()
}

type BargeInWithParam struct {
	link    StepLink
	applyFn BargeInApplyFunc
	marker  subroutineMarker
}

func (v BargeInWithParam) IsZero() bool {
	return v.applyFn == nil
}

func (v BargeInWithParam) IsValid() bool {
	return v.link.IsAtStep()
}

func (v BargeInWithParam) StepLink() StepLink {
	return v.link
}

/* ------ BargeIn support -------------------------- */

func (m *SlotMachine) createBargeIn(link StepLink, applyFn BargeInApplyFunc) BargeInWithParam {
	link.s.slotFlags |= slotHadBargeIn
	return BargeInWithParam{
		link:    link,
		applyFn: applyFn,
		marker:  link.s.getSubroutineMarker(),
	}
}

func (v BargeInWithParam) CallWithParam(a interface{}) bool {
	if !v.link.IsValid() { // if step was changed - will call anyway
		return false
	}
	m := v.link.getActiveMachine()
	if m == nil {
		return false
	}
	resultFn := v.applyFn(a)
	if resultFn == nil {
		return true
	}

	return m.executeBargeIn(v.link, DetachableSlotWorker{}, func(slot *Slot, w DetachableSlotWorker) StateUpdate {
		return v._execute(slot, w, resultFn)
	})
}

func (v BargeInWithParam) callInline(m *SlotMachine, param interface{}, worker DetachableSlotWorker) (done bool) {
	switch {
	case m == nil:
		panic(throw.IllegalValue())
	case worker.IsZero():
		panic(throw.IllegalValue())
	case !v.link.IsValid():
		// don't check IsAtStep() here as if step was changed we should do the call anyway
		return false
	}

	resultFn := v.applyFn(param)
	if resultFn == nil {
		return true
	}

	return m.executeBargeIn(v.link, worker, func(slot *Slot, w DetachableSlotWorker) StateUpdate {
		return v._execute(slot, w, resultFn)
	})
}

func (m *SlotMachine) createLightBargeIn(link StepLink, stateUpdate StateUpdate) BargeIn {
	if stateUpdate.link != nil || stateUpdate.param0 != 0 {
		panic(throw.IllegalValue())
	}

	link.s.slotFlags |= slotHadBargeIn
	h := BargeIn{stateUpdate, link.s.getSubroutineMarker()}

	// reuse internal fields
	h.update.link = link.s
	h.update.param0 = uint32(link.id)
	h.update.marker = ContextMarker(link.step)

	return h
}

func (v BargeIn) Call() bool {
	link := v.StepLink()
	m := link.getActiveMachine()
	if m == nil {
		return false
	}
	return m.executeBargeIn(link, DetachableSlotWorker{}, v._execute)
	//return v._callAsync(m, link)
}

func (v BargeIn) CallWithParam(param interface{}) bool {
	if param != nil {
		panic(throw.IllegalValue())
	}
	return v.Call()
}

func (v BargeIn) callInline(m *SlotMachine, worker DetachableSlotWorker) bool {
	switch {
	case m == nil:
		panic(throw.IllegalValue())
	case worker.IsZero():
		panic(throw.IllegalValue())
	}

	link := v.StepLink()
	if !link.IsAtStep() {
		return false
	}
	return m.executeBargeIn(link, worker, v._execute)
}

func (v BargeIn) _execute(slot *Slot, _ DetachableSlotWorker) StateUpdate {
	link := v.StepLink()
	if !link.IsAtStep() {
		return StateUpdate{}
	}
	stateUpdate := v.getStateUpdate()
	return slot.forceSubroutineUpdate(stateUpdate, v.marker)
}

func (m *SlotMachine) executeBargeIn(link StepLink, worker DetachableSlotWorker,
	executeFn func(slot *Slot, w DetachableSlotWorker) StateUpdate,
) bool {
	if !worker.IsZero() {
		switch tm := link.getActiveMachine(); {
		case tm == m:
			done := false
			m.applyAsyncCallback(link.SlotLink, wasInlineExec, worker,
				func(slot *Slot, worker DetachableSlotWorker, _ error) StateUpdate {
					done = true
					return executeFn(slot, worker)
				}, nil)
			if done {
				return true
			}
		case tm == nil:
			return false
		default:
			done := false
			worker.DetachableOuterCall(tm, func(worker DetachableSlotWorker) {
				tm.applyAsyncCallback(link.SlotLink, wasInlineExec, worker,
					func(slot *Slot, worker DetachableSlotWorker, _ error) StateUpdate {
						done = true
						return executeFn(slot, worker)
					}, nil)
			})
			if done {
				return true
			}
		}
	}

	return m.queueAsyncCallback(link.SlotLink, func(slot *Slot, worker DetachableSlotWorker, _ error) StateUpdate {
		return executeFn(slot, worker)
	}, nil)
}

func (v BargeInWithParam) _execute(slot *Slot, w DetachableSlotWorker, resultFn BargeInCallbackFunc) StateUpdate {
	_, isValid := slot.checkSubroutineMarker(v.marker)
	if !isValid {
		return StateUpdate{}
	}

	_, atExactStep := v.link.isValidAndAtExactStep()
	bc := bargingInContext{slotContext{s: slot, w: w}, atExactStep}
	stateUpdate := bc.executeBargeIn(resultFn)

	return slot.forceSubroutineUpdate(stateUpdate, v.marker)
}

func (m *SlotMachine) executeBargeInDirect(link StepLink, fn BargeInCallbackFunc, worker FixedSlotWorker) bool {
	if !m._canCallback(link.SlotLink) {
		return false
	}
	slot, isStarted, prevStepNo := link.tryStartWorking()
	if !isStarted {
		return false
	}

	needsStop := true
	defer func() {
		if needsStop {
			m.stopSlotWorking(slot, prevStepNo, worker)
		}
	}()

	slot.touchAfterInactive()

	_, atExactStep := link.isValidAndAtExactStep()
	bc := bargingInContext{slotContext{s: slot, w: worker.asDetachable()}, atExactStep}
	stateUpdate := bc.executeBargeInDirect(fn)
	stateUpdate = slot.forceTopSubroutineUpdate(stateUpdate)
	needsStop = false

	m.slotPostExecution(slot, stateUpdate, worker, prevStepNo, wasAsyncExec)

	return true
}
