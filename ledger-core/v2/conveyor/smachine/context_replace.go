/*
 * Copyright 2020 Insolar Network Ltd.
 * All rights reserved.
 * This material is licensed under the Insolar License version 1.0,
 * available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package smachine

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func (p *executionContext) Replace(fn CreateFunc) StateUpdate {
	return p.replaceExt(fn, CreateDefaultValues{})
}

func (p *executionContext) ReplaceExt(fn CreateFunc, defValues CreateDefaultValues) StateUpdate {
	return p.replaceExt(fn, defValues)
}

func (p *executionContext) replaceExt(fn CreateFunc, defValues CreateDefaultValues) StateUpdate {
	tmpl := p.template(stateUpdReplace) // ensures state of this context
	if fn == nil {
		panic(throw.IllegalValue())
	}

	setupFn := p.s.prepareReplaceWith(nil, fn, defValues)
	return tmpl.newStepOnly(SlotStep{Transition: setupFn})
}

func (p *executionContext) ReplaceWith(sm StateMachine) StateUpdate {
	tmpl := p.template(stateUpdReplace) // ensures state of this context
	if sm == nil {
		panic(throw.IllegalValue())
	}

	setupFn := p.s.prepareReplaceWith(sm, nil, CreateDefaultValues{})
	return tmpl.newStepOnly(SlotStep{Transition: setupFn})
}

func (s *Slot) prepareReplaceWith(sm StateMachine, fn CreateFunc, defValues CreateDefaultValues) StateFunc {
	if initFn := s.prepareSlotInit(s, fn, sm, defValues); initFn != nil {
		return initFn.defaultInit
	}
	panic("replacing SM didn't initialize")
}

func (p *executionContext) CallSubroutine(ssm SubroutineStateMachine, migrateFn MigrateFunc, exitFn SubroutineExitFunc) StateUpdate {
	p.ensure(updCtxExec)
	nextStep := p.s.prepareSubroutineStart(ssm, exitFn, migrateFn)
	return p.template(stateUpdSubroutineStart).newStepOnly(nextStep)
}
