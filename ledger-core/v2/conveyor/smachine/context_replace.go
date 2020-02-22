/*
 * Copyright 2020 Insolar Network Ltd.
 * All rights reserved.
 * This material is licensed under the Insolar License version 1.0,
 * available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.
 */

package smachine

import "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"

func (p *slotContext) _prepareReplacementData() prepareSlotValue {
	return prepareSlotValue{
		slotReplaceData: p.s.slotReplaceData.takeOutForReplace(),
		isReplacement:   true,
		tracerId:        p.s.getTracerId(),
	}
}

func (p *slotContext) Replace(fn CreateFunc) StateUpdate {
	tmpl := p.template(stateUpdReplace) // ensures state of this context

	if fn == nil {
		panic(throw.IllegalValue())
	}

	def := prepareReplaceData{fn: fn,
		def: p._prepareReplacementData(),
	}
	return tmpl.newVar(def)
}

func (p *slotContext) ReplaceExt(fn CreateFunc, defValues CreateDefaultValues) StateUpdate {
	tmpl := p.template(stateUpdReplace) // ensures state of this context

	if fn == nil {
		panic(throw.IllegalValue())
	}

	def := prepareReplaceData{fn: fn,
		def: p._prepareReplacementData(),
	}
	mergeDefaultValues(&def.def, defValues)

	return tmpl.newVar(def)
}

func (p *slotContext) ReplaceWith(sm StateMachine) StateUpdate {
	tmpl := p.template(stateUpdReplaceWith) // ensures state of this context

	if sm == nil {
		panic(throw.IllegalValue())
	}

	def := prepareReplaceData{sm: sm,
		def: p._prepareReplacementData(),
	}
	return tmpl.newVar(def)
}

func (p *executionContext) CallSubroutine(ssm SubroutineStateMachine, migrateFn MigrateFunc, exitFn SubroutineExitFunc) StateUpdate {
	p.ensure(updCtxExec)
	nextStep := p.s.prepareSubroutineStart(ssm, exitFn, migrateFn)
	return p.template(stateUpdSubroutineStart).newStepOnly(nextStep)
}
