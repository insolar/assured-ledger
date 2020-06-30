// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package requests

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/datawriter"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/inspectsvc"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.StateMachine = &SMRegisterRecordSet{}

type SMRegisterRecordSet struct {
	smachine.StateMachineDeclTemplate

	// input
	recordSet inspectsvc.RegisterRequestSet

	// injected
	pulseSlot  *conveyor.PulseSlot
	cataloger  datawriter.LineCataloger
	inspectSvc *inspectsvc.Adapter

	// runtime
	sdl          datawriter.LineDataLink
	inspectedSet *inspectsvc.InspectedRecordSet
	hasMissings  bool

	// results
	updated     *buildersvc.Future
}

func (p *SMRegisterRecordSet) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMRegisterRecordSet) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMRegisterRecordSet) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector *injector.DependencyInjector) {
	injector.MustInject(&p.pulseSlot)
	injector.MustInject(&p.cataloger)
	injector.MustInject(&p.inspectSvc)
}

func (p *SMRegisterRecordSet) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	switch {
	case p.pulseSlot.State() != conveyor.Present:
		return ctx.Error(throw.E("not a present pulse"))
	case p.recordSet.IsEmpty():
		return ctx.Error(throw.E("empty record set"))
	}

	ctx.SetDefaultMigration(p.migrate)
	ctx.SetDefaultErrorHandler(p.handleError)
	return ctx.Jump(p.stepFindLine)
}

func (p *SMRegisterRecordSet) stepFindLine(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.sdl.IsZero() {
		lineRef := p.getRootRef()

		p.sdl = p.cataloger.GetOrCreate(ctx, lineRef)
		if p.sdl.IsZero() {
			panic(throw.IllegalState())
		}
	}

	var limiter smachine.SyncLink
	switch p.sdl.TryAccess(ctx, func(sd *datawriter.LineSharedData) (wakeup bool) {
		limiter = sd.GetLimiter()
		return false
	}) {
	case smachine.Passed:
		//
	case smachine.NotPassed:
		return ctx.WaitShared(p.sdl.SharedDataLink).ThenRepeat()
	default:
		panic(throw.IllegalState())
	}

	if ctx.Acquire(limiter) {
		return ctx.Jump(p.stepLineIsReady)
	}

	return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if ctx.Acquire(limiter) {
			return ctx.Jump(p.stepLineIsReady)
		}
		return ctx.Sleep().ThenRepeat()
	})
}

func (p *SMRegisterRecordSet) stepLineIsReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	isValid := false
	switch p.sdl.TryAccess(ctx, func(sd *datawriter.LineSharedData) (wakeup bool) {
		isValid = sd.IsValid()
		return false
	}) {
	case smachine.Passed:
		//
	case smachine.NotPassed:
		return ctx.WaitShared(p.sdl.SharedDataLink).ThenRepeat()
	default:
		panic(throw.IllegalState())
	}

	if !isValid {
		return ctx.Jump(p.stepSendResponse)
	}

	// do chaining, hashing and signing via adapter
	return p.inspectSvc.PrepareInspectRecordSet(ctx, p.recordSet, func(ctx smachine.AsyncResultContext, set inspectsvc.InspectedRecordSet, err error) {
		if err != nil {
			panic(err)
		}
		p.inspectedSet = &set
	}).DelayedStart().Sleep().ThenJump(p.stepApplyRecordSet)
}

func (p *SMRegisterRecordSet) stepApplyRecordSet(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.inspectedSet == nil {
		return ctx.Sleep().ThenRepeat()
	}

	switch p.sdl.TryAccess(ctx, func(sd *datawriter.LineSharedData) (wakeup bool) {
		switch future, missings := sd.TryApplyRecordSet(*p.inspectedSet); {
		case missings == nil:
			p.updated = future
			return false
		case p.hasMissings:
			return false
		default:
			p.hasMissings = true
			sd.RequestMissings(missings, ctx.NewBargeIn().WithWakeUp())
			return true
		}
	}) {
	case smachine.Passed:
		//
	case smachine.NotPassed:
		return ctx.WaitShared(p.sdl.SharedDataLink).ThenRepeat()
	default:
		panic(throw.IllegalState())
	}

	if p.updated == nil {
		return ctx.Sleep().ThenRepeat()
	}

	if p.getFlags() & rms.RegistrationFlags_Fast != 0 {
		p.sendResponse(false, true)
	}

	return ctx.Jump(p.stepWaitUpdated)
}

func (p *SMRegisterRecordSet) stepWaitUpdated(ctx smachine.ExecutionContext) smachine.StateUpdate {
	ctx.ReleaseAll()

	if !ctx.Acquire(p.updated.GetReadySync()) {
		return ctx.Sleep().ThenRepeat()
	}
	return ctx.Jump(p.stepSendResponse)
}

func (p *SMRegisterRecordSet) migrate(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(nil)
	return ctx.Jump(p.stepSendResponse)
}

func (p *SMRegisterRecordSet) stepSendResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	p.sendResponse(true, p.updated != nil && p.updated.IsCommitted())
	return ctx.Stop()
}

func (p *SMRegisterRecordSet) sendResponse(safe, ok bool) {
	// TODO
	_, _ = safe, ok
	panic(throw.NotImplemented())
}

func (p *SMRegisterRecordSet) getRootRef() reference.Global {
	return p.recordSet.Excerpts[0].RootRef.GetGlobal()
}

func (p *SMRegisterRecordSet) getFlags() rms.RegistrationFlags {
	return p.recordSet.Requests[0].Flags
}

func (p *SMRegisterRecordSet) handleError(smachine.FailureContext) {
	p.sendResponse(true, false)
}

