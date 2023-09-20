package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc/bundle"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/dataextractor"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewDirtyReader(cfg dataextractor.Config) smachine.SubroutineStateMachine {
	cfg.Ensure()
	return &SubSMDirtyReader{cfg: cfg}
}

var _ smachine.SubroutineStateMachine = &SubSMDirtyReader{}
type SubSMDirtyReader struct {
	smachine.StateMachineDeclTemplate

	// input
	cfg dataextractor.Config

	// injected
	cataloger LineCataloger
	reader    buildersvc.ReadAdapter

	// runtime
	sdl     LineDataLink
	dropID  jet.DropID
	selfRef reference.Local

	sequence dirtyIterator
}

func (p *SubSMDirtyReader) GetStateMachineDeclaration() smachine.StateMachineDeclaration {
	return p
}

func (p *SubSMDirtyReader) InjectDependencies(_ smachine.StateMachine, _ smachine.SlotLink, injector injector.DependencyInjector) {
	injector.MustInject(&p.cataloger)
	injector.MustInject(&p.reader)
}

func (p *SubSMDirtyReader) GetSubroutineInitState(smachine.SubroutineStartContext) smachine.InitFunc {
	return p.stepInit
}

func (p *SubSMDirtyReader) GetInitStateFor(smachine.StateMachine) smachine.InitFunc {
	return p.stepInit
}

func (p *SubSMDirtyReader) stepInit(ctx smachine.InitializationContext) smachine.StateUpdate {
	// switch {
	// case p.request.TargetStartRef.IsEmpty():
	// 	return ctx.Error(throw.E("missing target"))
	// case p.request.Flags != 0:
	// 	panic(throw.NotImplemented())
	// }
	//
	// if tpn := p.request.TargetStartRef.Get().GetLocal().Pulse(); tpn != p.pn {
	// 	return ctx.Error(throw.E("wrong target pulse", struct { TargetPN, SlotPN pulse.Number }{ tpn, p.pn}))
	// }

	// p.request.LimitCount
	// p.request.LimitSize
	// p.request.LimitRef
	// p.request.Flags

	switch notReason, selectorRef := p.cfg.Selector.GetSelectorRef(); {
	case notReason:
		p.selfRef = reference.NormCopy(selectorRef).GetBase()
	default:
		panic(throw.NotImplemented())
	}

	return ctx.Jump(p.stepFindLine)
}

func (p *SubSMDirtyReader) stepFindLine(ctx smachine.ExecutionContext) smachine.StateUpdate {
	if p.sdl.IsZero() {
		lineRef := reference.NewSelf(p.selfRef)

		p.sdl = p.cataloger.Get(ctx, lineRef)
		if p.sdl.IsZero() {
			// line is absent
			return ctx.Jump(p.stepResponse)
		}
	}

	var activator smachine.SyncLink
	switch p.sdl.TryAccess(ctx, func(sd *LineSharedData) (wakeup bool) {
		activator = sd.GetActiveSync()
		return false
	}) {
	case smachine.Passed:
		//
	case smachine.NotPassed:
		return ctx.WaitShared(p.sdl.SharedDataLink).ThenRepeat()
	default:
		panic(throw.IllegalState())
	}

	if ctx.Acquire(activator) {
		ctx.ReleaseAll()
		return ctx.Jump(p.stepLineIsReady)
	}

	return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if ctx.Acquire(activator) {
			ctx.ReleaseAll()
			return ctx.Jump(p.stepLineIsReady)
		}
		return ctx.Sleep().ThenRepeat()
	})
}

func (p *SubSMDirtyReader) stepLineIsReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	var future *buildersvc.Future
	valid := false

	switch p.sdl.TryAccess(ctx, func(sd *LineSharedData) (wakeup bool) {
		if !sd.IsValidForRead() {
			return
		}
		valid = true
		p.dropID = sd.DropID()
		sd.TrimStages()

		_, future = p.findExpectedRecords(sd)

		return false
	}) {
	case smachine.Passed:
		//
	case smachine.NotPassed:
		return ctx.WaitShared(p.sdl.SharedDataLink).ThenRepeat()
	default:
		panic(throw.IllegalState())
	}

	switch {
	case !valid:
		return ctx.Jump(p.stepResponse)
	case future == nil:
		return ctx.Jump(p.stepPrepareDataWithReader)
	}

	ready := future.GetReadySync()

	if ctx.Acquire(ready) {
		ctx.ReleaseAll()
		return ctx.Jump(p.stepDataIsReady)
	}

	return ctx.Sleep().ThenJump(func(ctx smachine.ExecutionContext) smachine.StateUpdate {
		if ctx.Acquire(ready) {
			ctx.ReleaseAll()
			return ctx.Jump(p.stepDataIsReady)
		}
		return ctx.Sleep().ThenRepeat()
	})
}

func (p *SubSMDirtyReader) stepDataIsReady(ctx smachine.ExecutionContext) smachine.StateUpdate {
	switch p.sdl.TryAccess(ctx, func(sd *LineSharedData) (wakeup bool) {
		sd.TrimStages()

		if _, future := p.findExpectedRecords(sd); future != nil {
			panic(throw.Impossible())
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

	return ctx.Jump(p.stepPrepareDataWithReader)
}

func (p *SubSMDirtyReader) stepPrepareDataWithReader(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return p.reader.PrepareAsync(ctx, func(svc buildersvc.ReadService) smachine.AsyncResultFunc {
		extractor := dataextractor.NewSequenceReader(&p.sequence, p.cfg.Limiter, p.cfg.Output)

		err := svc.DropReadDirty(p.dropID, func(reader bundle.DirtyReader) error {
			extractor.SetReader(reader)
			return extractor.ReadAll()
		})

		return func(ctx smachine.AsyncResultContext) {
			if err != nil {
				panic(err)
			}
			ctx.WakeUp()
		}
	}).DelayedStart().Sleep().ThenJump(p.stepResponse)
}

func (p *SubSMDirtyReader) stepResponse(ctx smachine.ExecutionContext) smachine.StateUpdate {
	return ctx.Stop()
}

func (p *SubSMDirtyReader) findExpectedRecords(sd *LineSharedData) (bool, *buildersvc.Future) {
	// TODO do a few sub-cycles when too many record

	lim := p.cfg.Limiter.Clone()

	return sd.FindSequence(p.cfg.Selector, func(record lineage.ReadRecord) bool {
		lim.Next(0, record.RecRef)
		if lim.CanRead() {
			p.sequence.addExpected(record.StorageIndex)
			return false
		}
		return true
	})
}
