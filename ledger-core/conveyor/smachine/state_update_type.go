// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func newStateUpdateTemplate(contextType updCtxMode, marker ContextMarker, updKind stateUpdKind) StateUpdateTemplate {
	return stateUpdateTypes[updKind].template(marker, contextType)
}

func _getStateUpdateType(updKind stateUpdKind) (StateUpdateType, bool) {
	if int(updKind) >= len(stateUpdateTypes) {
		return StateUpdateType{}, false
	}
	sut := stateUpdateTypes[updKind]
	if sut.canGet() {
		return sut, true
	}
	return StateUpdateType{}, false
}

// func getStateUpdateType(stateUpdate StateUpdate) (StateUpdateType, bool) {
// 	return _getStateUpdateType(stateUpdKind(stateUpdate.updKind))
// }

func getStateUpdateTypeAndName(stateUpdate StateUpdate) (StateUpdateType, string, bool) {
	if sut, ok := _getStateUpdateType(stateUpdKind(stateUpdate.updKind)); ok {
		if len(sut.name) > 0 {
			return sut, sut.name, true
		}
		return sut, fmt.Sprintf("noname(%d)", stateUpdate.updKind), true
	}
	return StateUpdateType{}, fmt.Sprintf("unknown(%d)", stateUpdate.updKind), false
}

func typeOfStateUpdate(stateUpdate StateUpdate) StateUpdateType {
	return stateUpdateTypes[stateUpdate.updKind].get()
}

func typeOfStateUpdateForPrepare(contextMode updCtxMode, stateUpdate StateUpdate) StateUpdateType {
	return stateUpdateTypes[stateUpdate.updKind].getForPrepare(contextMode)
}

func newPanicStateUpdate(err error) StateUpdate {
	return StateUpdateTemplate{t: &stateUpdateTypes[stateUpdPanic]}.newError(err)
}

func recoverSlotPanicAsUpdate(update StateUpdate, msg string, recovered interface{}, prev error, area SlotPanicArea) StateUpdate {
	switch {
	case recovered != nil:
		return newPanicStateUpdate(RecoverSlotPanicWithStack(msg, recovered, prev, area))
	case prev != nil:
		return newPanicStateUpdate(prev)
	}
	return update
}

// func getStateUpdateKind(stateUpdate StateUpdate) stateUpdKind {
// 	return stateUpdKind(stateUpdate.updKind)
// }

type SlotUpdateFunc func(slot *Slot, stateUpdate StateUpdate, worker FixedSlotWorker, sd *StepDeclaration) (isAvailable bool, err error)
type SlotUpdatePrepareFunc func(slot *Slot, stateUpdate *StateUpdate)
type SlotUpdateShortLoopFunc func(slot *Slot, stateUpdate StateUpdate, loopCount uint32, sd *StepDeclaration) bool

type StateUpdateType struct {
	updKind stateUpdKind

	/* Runs within a valid ExecutionContext and under detachable worker */
	prepare SlotUpdatePrepareFunc
	/* Runs without a valid ExecutionContext and under detachable worker */
	shortLoop SlotUpdateShortLoopFunc
	/* Runs under non-detachable worker */
	apply SlotUpdateFunc

	varVerify func(interface{})

	name string

	filter             updCtxMode
	params             stateUpdParam
	stepDeclaration    *StepDeclaration
	safeWithSubroutine bool
}

type StateUpdateTemplate struct {
	t *StateUpdateType

	marker  ContextMarker
	ctxType updCtxMode
}

type stateUpdBaseType = uint8
type stateUpdKind stateUpdBaseType
type stateUpdParam uint8
type updCtxMode uint32

const updCtxInactive updCtxMode = 0

const (
	updCtxDiscarded updCtxMode = 1 << iota
	updCtxInternal             // special mode - updates can't be accessed via template() call, but getForXXX() allows any valid context
	updCtxMachineCall
	updCtxFail

	updCtxBargeIn
	updCtxSubrExit
	updCtxAsyncCallback

	updCtxConstruction
	updCtxFinalize
	updCtxSubrStart
	updCtxInit
	updCtxExec
	updCtxMigrate
)

const (
	updParamStep stateUpdParam = 1 << iota
	updParamUint
	updParamLink
	updParamVar
)

func (v StateUpdateType) verify(ctxType updCtxMode, allowInternal bool) {
	switch {
	case ctxType <= updCtxDiscarded:
		panic(v.panicText(ctxType, "illegal value"))
	case v.apply == nil:
		panic(v.panicText(ctxType, "not implemented"))

	case ctxType&v.filter == ctxType:
		return
	case allowInternal && v.filter&updCtxInternal != 0:
		return
	default:
		panic(v.panicText(ctxType, "not allowed"))
	}
}

func (v StateUpdateType) panicText(ctxType updCtxMode, msg string) string {
	return fmt.Sprintf("updKind=%v ctxType=%v: %v", v.updKind, ctxType, msg)
}

func (v StateUpdateType) template(marker ContextMarker, ctxType updCtxMode) StateUpdateTemplate {
	v.verify(ctxType, false)
	return StateUpdateTemplate{&v, marker, ctxType}
}

func (v StateUpdateType) getForPrepare(ctxType updCtxMode) StateUpdateType {
	v.verify(ctxType, true)
	return v
}

func (v StateUpdateType) get() StateUpdateType {
	if v.apply == nil {
		panic("not implemented")
	}
	return v
}

func (v StateUpdateType) canGet() bool {
	return v.apply != nil
}

func (v StateUpdateType) verifyVar(u interface{}) interface{} {
	if v.varVerify != nil {
		v.varVerify(u)
	}
	return u
}

func (v StateUpdateType) ShortLoop(slot *Slot, stateUpdate StateUpdate, loopCount uint32) bool {
	if v.shortLoop == nil {
		return false
	}
	return v.shortLoop(slot, stateUpdate, loopCount, v.stepDeclaration)
}

func (v StateUpdateType) Prepare(slot *Slot, stateUpdate *StateUpdate) {
	if v.prepare != nil {
		v.prepare(slot, stateUpdate)
	}
}

func (v StateUpdateType) Apply(slot *Slot, stateUpdate StateUpdate, worker FixedSlotWorker) (isAvailable bool, err error) {
	if v.apply == nil {
		return false, throw.NotImplemented()
	}
	return v.apply(slot, stateUpdate, worker, v.stepDeclaration)
}

func (v StateUpdateType) IsSubroutineSafe() bool {
	return v.safeWithSubroutine
}

func (v StateUpdateType) GetStepDeclaration() *StepDeclaration {
	return v.stepDeclaration
}

func (v StateUpdateTemplate) ensureTemplate(params stateUpdParam) {
	if v.t == nil {
		panic("illegal state")
	}
	if v.t.params&params != params {
		panic("illegal value")
	}
	if v.t.apply == nil {
		panic("illegal kind")
	}
}

func (v StateUpdateTemplate) newNoArg() StateUpdate {
	v.ensureTemplate(0)
	return StateUpdate{
		marker:  v.marker,
		updKind: stateUpdBaseType(v.t.updKind),
	}
}

type StepPrepareFunc func()

func (v StateUpdateTemplate) newNonNilStep(slotStep SlotStep) StateUpdate {
	if slotStep.Transition == nil {
		panic(throw.IllegalValue())
	}
	return v.newStep(slotStep, nil)
}

func (v StateUpdateTemplate) newStep(slotStep SlotStep, prepare StepPrepareFunc) StateUpdate {
	v.ensureTemplate(updParamStep | updParamVar)
	return StateUpdate{
		marker:  v.marker,
		updKind: stateUpdBaseType(v.t.updKind),
		step:    slotStep,
		param1:  prepare,
	}
}

func (v StateUpdateTemplate) newStepOnly(slotStep SlotStep) StateUpdate {
	v.ensureTemplate(updParamStep)
	if slotStep.Transition == nil {
		panic(throw.IllegalValue())
	}
	return StateUpdate{
		marker:  v.marker,
		updKind: stateUpdBaseType(v.t.updKind),
		step:    slotStep,
	}
}

func (v StateUpdateTemplate) newStepUntil(slotStep SlotStep, prepare StepPrepareFunc, until uint32) StateUpdate {
	v.ensureTemplate(updParamStep | updParamUint | updParamVar)
	return StateUpdate{
		marker:  v.marker,
		updKind: stateUpdBaseType(v.t.updKind),
		step:    slotStep,
		param1:  prepare,
		param0:  until,
	}
}

func (v StateUpdateTemplate) newStepUint(slotStep SlotStep, param uint32) StateUpdate {
	v.ensureTemplate(updParamStep | updParamUint)
	return StateUpdate{
		marker:  v.marker,
		updKind: stateUpdBaseType(v.t.updKind),
		param0:  param,
		step:    slotStep,
	}
}

func (v StateUpdateTemplate) newStepLink(slotStep SlotStep, link SlotLink) StateUpdate {
	v.ensureTemplate(updParamStep | updParamLink)
	return StateUpdate{
		marker:  v.marker,
		updKind: stateUpdBaseType(v.t.updKind),
		link:    link.s,
		param0:  uint32(link.id),
		step:    slotStep,
	}
}

// nolint:unused
func (v StateUpdateTemplate) newVar(u interface{}) StateUpdate {
	v.ensureTemplate(updParamVar)
	return StateUpdate{
		marker:  v.marker,
		updKind: stateUpdBaseType(v.t.updKind),
		param1:  v.t.verifyVar(u),
	}
}

func (v StateUpdateTemplate) newError(e error) StateUpdate {
	v.ensureTemplate(updParamVar)
	return StateUpdate{
		marker:  v.marker,
		updKind: stateUpdBaseType(v.t.updKind),
		param1:  v.t.verifyVar(e),
	}
}

func (v StateUpdateTemplate) newUint(param uint32) StateUpdate {
	v.ensureTemplate(updParamUint)
	return StateUpdate{
		marker:  v.marker,
		updKind: stateUpdBaseType(v.t.updKind),
		param0:  param,
	}
}
