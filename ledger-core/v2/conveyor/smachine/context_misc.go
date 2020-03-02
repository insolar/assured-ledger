// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ ConstructionContext = &constructionContext{}

type constructionContext struct {
	contextTemplate
	s            *Slot
	creator      *Slot
	injects      map[string]interface{}
	inherit      DependencyInheritanceMode
	tracerId     TracerId
	callbackFn   TerminationCallbackFunc
	callbackLink SlotLink
	isTracing    bool
}

func (p *constructionContext) SetDependencyInheritanceMode(mode DependencyInheritanceMode) {
	p.ensure(updCtxConstruction)
	p.inherit = mode
}

func (p *constructionContext) OverrideDependency(id string, v interface{}) {
	p.ensure(updCtxConstruction)
	if p.injects == nil {
		p.injects = make(map[string]interface{})
	}
	p.injects[id] = v
}

func (p *constructionContext) SlotLink() SlotLink {
	p.ensure(updCtxConstruction)
	return p.s.NewLink()
}

func (p *constructionContext) GetContext() context.Context {
	p.ensure(updCtxConstruction)
	return p.s.ctx
}

func (p *constructionContext) SetContext(ctx context.Context) {
	p.ensure(updCtxConstruction)
	if ctx == nil {
		panic("illegal value")
	}
	p.s.ctx = ctx
}

func (p *constructionContext) ParentLink() SlotLink {
	p.ensure(updCtxConstruction)
	return p.s.parent
}

func (p *constructionContext) SetParentLink(parent SlotLink) {
	p.ensure(updCtxConstruction)
	p.s.parent = parent
}

func (p *constructionContext) SetTerminationCallback(parentCtx ExecutionContext, callbackFn TerminationCallbackFunc) {
	p.ensure(updCtxConstruction)
	if callbackFn == nil {
		if parentCtx != nil {
			parentCtx.SlotLink() // to validate
		}
		p.callbackFn = nil
		p.callbackLink = SlotLink{}
		return
	}
	if ec, ok := parentCtx.(*executionContext); !ok || ec.s != p.creator {
		panic(throw.IllegalValue())
	}
	p.callbackLink = parentCtx.SlotLink()
	p.callbackFn = callbackFn
}

func (p *constructionContext) SetDefaultTerminationResult(v interface{}) {
	p.ensure(updCtxConstruction)
	p.s.defResult = v
}

func (p *constructionContext) SetLogTracing(isTracing bool) {
	p.ensure(updCtxConstruction)
	p.isTracing = isTracing
}

func (p *constructionContext) SetTracerId(tracerId TracerId) {
	p.ensure(updCtxConstruction)
	p.tracerId = tracerId
}

func (p *constructionContext) executeCreate(nextCreate CreateFunc) StateMachine {
	p.setMode(updCtxConstruction)
	defer p.setDiscarded()

	return nextCreate(p)
}

/* ========================================================================= */

var _ InitializationContext = &initializationContext{}

type initializationContext struct {
	slotContext
}

func (p *initializationContext) executeInitialization(fn InitFunc) (stateUpdate StateUpdate) {
	p.setMode(updCtxInit)
	defer func() {
		stateUpdate = p.discardAndUpdate("initialization", recover(), stateUpdate, StateArea)
	}()

	return p.ensureAndPrepare(p.s, fn(p))
}

/* ========================================================================= */

var _ MigrationContext = &migrationContext{}

type migrationContext struct {
	slotContext
	fixedWorker  FixedSlotWorker
	skipMultiple bool
}

func (p *migrationContext) SkipMultipleMigrations() {
	p.ensure(updCtxMigrate)
	p.skipMultiple = true
}

func (p *migrationContext) executeMigration(fn MigrateFunc) (stateUpdate StateUpdate, skipMultiple bool) {
	p.setMode(updCtxMigrate)
	defer func() {
		stateUpdate = p.discardAndUpdate("migration", recover(), stateUpdate, StateArea)
	}()

	su := p.ensureAndPrepare(p.s, fn(p))
	return su, p.skipMultiple
}

/* ========================================================================= */

var _ FailureContext = &failureContext{}

type failureContext struct {
	slotContext
	isPanic    bool
	area       SlotPanicArea
	canRecover bool
	err        error
	result     interface{}
	action     ErrorHandlerAction
}

func (p *failureContext) GetDefaultTerminationResult() interface{} {
	p.ensure(updCtxFail)
	return p.s.defResult
}

func (p *failureContext) SetTerminationResult(v interface{}) {
	p.ensure(updCtxFail)
	p.result = v
}

func (p *failureContext) GetError() error {
	p.ensure(updCtxFail)
	return p.err
}

func (p *failureContext) IsPanic() bool {
	p.ensure(updCtxFail)
	return p.isPanic
}

func (p *failureContext) GetArea() SlotPanicArea {
	p.ensure(updCtxFail)
	return p.area
}

func (p *failureContext) CanRecover() bool {
	p.ensure(updCtxFail)
	return p.canRecover
}

func (p *failureContext) SetAction(action ErrorHandlerAction) {
	p.ensure(updCtxFail)
	p.action = action
}

func (p *failureContext) executeFailure(fn ErrorHandlerFunc) (ok bool, result ErrorHandlerAction, err error) {
	p.setMode(updCtxFail)
	defer func() {
		p.discardAndCapture("failure handler", recover(), &err, ErrorHandlerArea)
	}()
	err = p.err // ensure it will be included on panic
	fn(p)
	return true, p.action, err
}
