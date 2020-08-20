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
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ smachine.StateMachine = &SMRegisterRecordSet{}

func NewSMRegisterRecordSet(reqs inspectsvc.RegisterRequestSet) *SMRegisterRecordSet {
	return &SMRegisterRecordSet{
		recordSet: reqs,
	}
}

type SMRegisterRecordSet struct {
	smachine.StateMachineDeclTemplate

	// input
	recordSet inspectsvc.RegisterRequestSet

	// injected
	pulseSlot  *conveyor.PulseSlot
	cataloger  datawriter.LineCataloger
	inspectSvc inspectsvc.Adapter

	// runtime
	sdl          datawriter.LineDataLink
	inspectedSet inspectsvc.InspectedRecordSet
	hasRequested bool
	isCommitted  bool

	// results
	committed *buildersvc.Future
}

func (p *SMRegisterRecordSet) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SMRegisterRecordSet) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SMRegisterRecordSet) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
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

	ctx.SetDefaultMigration(p.migratePresent)
	ctx.SetDefaultErrorHandler(p.handleError)

	p.recordSet.Validate() // panic will be handled by p.handleError

	return ctx.Jump(p.stepFindLine)
}

func (p *SMRegisterRecordSet) stepFindLine(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.sdl.IsZero() {
		lineRef := p.recordSet.GetRootRef()

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
		return ctx.Jump(p.stepSendFinalResponse)
	}

	// do chaining, hashing and signing via adapter
	return p.inspectSvc.PrepareInspectRecordSet(ctx, p.recordSet,
		func(ctx smachine.AsyncResultContext, set inspectsvc.InspectedRecordSet, err error) {
			if err != nil {
				panic(err)
			}
			p.inspectedSet = set
		},
	).DelayedStart().Sleep().ThenJump(p.stepApplyRecordSet)
}

func (p *SMRegisterRecordSet) stepApplyRecordSet(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.inspectedSet.Records == nil {
		return ctx.Sleep().ThenRepeat()
	}

	var errors []error
	committedDuplicates := false

	switch p.sdl.TryAccess(ctx, func(sd *datawriter.LineSharedData) (wakeup bool) {
		switch future, bundle := sd.TryApplyRecordSet(ctx, p.inspectedSet); {
		case future != nil:
			if bundle != nil {
				panic(throw.Impossible())
			}
			sd.CollectSignatures(p.inspectedSet)
			p.committed = future
		case bundle == nil:
			committedDuplicates = true
			sd.CollectSignatures(p.inspectedSet)

		case bundle.HasErrors():
			errors = bundle.GetErrors()
		case p.hasRequested:
			//
		default:
			p.hasRequested = true
			sd.RequestDependencies(bundle, ctx.NewBargeIn().WithWakeUp())
			return true
		}
		return false
	}) {
	case smachine.Passed:
		//
	case smachine.NotPassed:
		return ctx.WaitShared(p.sdl.SharedDataLink).ThenRepeat()
	default:
		panic(throw.IllegalState())
	}

	sendUnsafeConfo := false
	switch {
	case len(errors) > 0:
		ctx.ReleaseAll()
		return p.handleFailure(ctx, errors...)

	case committedDuplicates:
		p.isCommitted = true

	case p.committed == nil:
		return ctx.Sleep().ThenRepeat()

	case p.recordSet.GetFlags() & rms.RegistrationFlags_Fast != 0:
		// this check is to avoid sending unsafe and safe responses simultaneously
		if !ctx.Acquire(p.committed.GetReadySync()) {
			sendUnsafeConfo = true
		} else {
			p.cleanup(ctx)
			p.isCommitted = true
		}
	}
	ctx.ReleaseAll()

	if p.isCommitted {
		return ctx.Jump(p.stepSendFinalResponse)
	}

	if sendUnsafeConfo {
		p.sendResponse(ctx, false)
	}

	return ctx.Jump(p.stepWaitCommitted)
}

func (p *SMRegisterRecordSet) stepWaitCommitted(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if !ctx.Acquire(p.committed.GetReadySync()) {
		return ctx.Sleep().ThenRepeat()
	}

	p.cleanup(ctx)

	return ctx.Jump(p.stepSendFinalResponse)
}

func (p *SMRegisterRecordSet) migratePresent(ctx smachine.MigrationContext) smachine.StateUpdate {
	ctx.SetDefaultMigration(p.migratePast)

	if p.committed == nil {
		// records were not pushed into the storage
		// probably we are waiting to resolve dependencies or just late
		return ctx.Jump(p.stepSendFinalResponse)
	}

	// have to wait for confirmation from storage
	return ctx.Stay()
}

func (p *SMRegisterRecordSet) migratePast(ctx smachine.MigrationContext) smachine.StateUpdate {
	return ctx.Stop()
}

func (p *SMRegisterRecordSet) stepSendFinalResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	switch {
	case p.committed != nil:
		switch ready, err := p.committed.GetFutureResult(); {
		case !ready:
			return p.handleFailure(ctx, throw.E("aborted"))
		case err != nil:
			return p.handleFailure(ctx, err)
		}
	case !p.isCommitted:
		return p.handleFailure(ctx, throw.E("cancelled"))
	}

	p.sendResponse(ctx, true)
	return ctx.Stop()
}

func (p *SMRegisterRecordSet) handleFailure(ctx smachine.ExecutionContext, errors ...error) smachine.StateUpdate {
	if len(errors) == 0 || errors[0] == nil {
		panic(throw.IllegalValue())
	}

	if p.sendFailResponse(ctx, errors...) {
		return ctx.Stop()
	}

	return ctx.Error(errors[0])
}

func (p *SMRegisterRecordSet) sendResponse(ctx smachine.ExecutionContext, safe bool) {
	signatures := make([]cryptkit.Signature, len(p.inspectedSet.Records))
	for i := range p.inspectedSet.Records {
		signatures[i] = p.inspectedSet.Records[i].RegistrarSignature.GetSignature()
	}

	if safe {
		ctx.SetDefaultTerminationResult(signatures)
	}

	// TODO implement send
}

func (p *SMRegisterRecordSet) sendFailResponse(smachine.FailureExecutionContext, ...error) bool {
	// TODO implement failure send
	return false
}

func (p *SMRegisterRecordSet) handleError(ctx smachine.FailureContext) {
	p.sendFailResponse(ctx, ctx.GetError())
}

func (p *SMRegisterRecordSet) cleanup(ctx smachine.ExecutionContext) {
	// don't worry when there is no access - it will be trimmed later then
	p.sdl.TryAccess(ctx, func(sd *datawriter.LineSharedData) (wakeup bool) {
		sd.TrimStages()
		return false
	})
}
