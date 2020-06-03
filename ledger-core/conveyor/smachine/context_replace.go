// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func (p *executionContext) Replace(fn CreateFunc) StateUpdate {
	tmpl := p.template(stateUpdReplace) // ensures state of this context
	if fn == nil {
		panic(throw.IllegalValue())
	}
	return tmpl.newStepOnly(p.prepareReplace(nil, fn, CreateDefaultValues{}))
}

func (p *executionContext) ReplaceExt(fn CreateFunc, defValues CreateDefaultValues) StateUpdate {
	tmpl := p.template(stateUpdReplace) // ensures state of this context
	if fn == nil {
		panic(throw.IllegalValue())
	}
	return tmpl.newStepOnly(p.prepareReplace(nil, fn, defValues))
}

func (p *executionContext) ReplaceWith(sm StateMachine) StateUpdate {
	tmpl := p.template(stateUpdReplace) // ensures state of this context
	if sm == nil {
		panic(throw.IllegalValue())
	}
	return tmpl.newStepOnly(p.prepareReplace(sm, nil, CreateDefaultValues{}))
}

func (p *executionContext) prepareReplace(sm StateMachine, fn CreateFunc, defValues CreateDefaultValues) SlotStep {
	return SlotStep{Transition: p.s.prepareReplace(fn, sm, defValues)}
}
