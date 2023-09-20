package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/dataextractor"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/inspectsvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type LineKey reference.Global

type LineSharedData struct {
	lineRef    reference.Global
	limiter    smsync.SemaphoreLink
	activeSync smsync.BoolConditionalLink

	ready bool
	valid bool

	jetDropID jet.DropID
	resolver  lineage.DependencyResolver
	adapter   buildersvc.WriteAdapter

	data             *lineage.LineStages
	deps             DependencyTracker
	dropFinalizeSync smachine.SyncLink
}

func (p *LineSharedData) LineRef() reference.Global {
	return p.lineRef
}

func (p *LineSharedData) GetLimiter() smachine.SyncLink {
	return p.limiter.SyncLink()
}

func (p *LineSharedData) GetActiveSync() smachine.SyncLink {
	switch {
	case !p.ready:
		return p.activeSync.SyncLink()
	case !p.activeSync.IsZero():
		p.activeSync = smsync.BoolConditionalLink{}
	}
	return everOpenSync
}

var everOpenSync = smsync.NewAlwaysOpen("SMLine.always-open")

func (p *LineSharedData) ensureDataAccess() {
	if !p.jetDropID.IsValid() {
		panic(throw.IllegalState())
	}
	if p.data == nil {
		p.ensureAccess()
		p.data = lineage.NewStages(p.lineRef.GetBase(), p.jetDropID.CreatedAt(), p.resolver)
	}
}

func (p *LineSharedData) TryApplyRecordSet(ctx smachine.ExecutionContext,
	set inspectsvc.InspectedRecordSet, verifyOnly bool,
) (*buildersvc.Future, *buildersvc.Future, *lineage.BundleResolver) {

	p.ensureDataAccess()
	if set.IsEmpty() {
		panic(throw.IllegalValue())
	}

	p.data.TrimCommittedStages()
	br := p.data.NewBundle()
	br.Hint(set.Count())

	for _, r := range set.Records {
		br.Add(r)
	}

	if verifyOnly {
		fut, br := p.verifyBundle(br)
		return nil, fut, br
	}

	return p.applyBundle(ctx, br)
}

func (p *LineSharedData) verifyBundle(br *lineage.BundleResolver) (*buildersvc.Future, *lineage.BundleResolver) {
	if !br.IsReadyForStage() {
		return nil, br
	}

	switch ok, fut := p.data.VerifyBundle(br); {
	case !ok:
		// got an error or an unresolved dependency, or a missing record
		return nil, br
	case fut == nil:
		// all entries were already added and committed
		return nil, nil
	default:
		return fut.(*buildersvc.Future), nil
	}
}

func (p *LineSharedData) applyBundle(ctx smachine.ExecutionContext, br *lineage.BundleResolver) (*buildersvc.Future, *buildersvc.Future, *lineage.BundleResolver) {

	if !br.IsReadyForStage() {
		return nil, nil, br
	}

	future := buildersvc.NewFuture("")

	switch result, fut, resolved := p.data.AddBundle(br, future); result {
	case lineage.UnableToAdd:
		// got an error or an unresolved dependency
		return nil, nil, br

	case lineage.WaitForHead:
		return fut.(*buildersvc.Future), nil, nil

	default:
		switch {
		case fut == nil:
			// all entries were already added and committed
			if !resolved.IsZero() {
				panic(throw.Impossible())
			}
			return nil, nil, nil

		case resolved.IsZero():
			// all entries were already added, but not yet committed
			return nil, fut.(*buildersvc.Future), nil

		default:
			// entries has to be added

			if future != fut {
				panic(throw.Impossible())
			}

			dropID := p.jetDropID
			p.adapter.PrepareNotify(ctx, func(service buildersvc.Service) {
				service.AppendToDrop(dropID, future, resolved)
			}).Send()

			return nil, future, nil
		}
	}
}

func (p *LineSharedData) RequestDependencies(br *lineage.BundleResolver, wakeup smachine.BargeInNoArgHolder) {
	p.ensureAccess()
	if p.data == nil {
		panic(throw.IllegalState())
	}

	deps := br.GetUnresolvedDependencies()
	if len(deps) == 0 {
		panic(throw.IllegalValue())
	}
	p.deps.AddDependencySet(deps, func() {
		wakeup.Call()
	})
}

func (p *LineSharedData) ensureAccess() {
	if !p.ready {
		panic(throw.IllegalState())
	}
}

func (p *LineSharedData) IsValid() bool {
	p.ensureAccess()
	return p.valid
}

func (p *LineSharedData) IsValidForRead() bool {
	return p.valid
}

func (p *LineSharedData) enableAccess() smachine.SyncAdjustment {
	p.ready = true
	return p.limiter.NewValue(1).MergeWith(p.activeSync.NewValue(true))
}

func (p *LineSharedData) disableAccess() {
	p.ready = false
}

func (p *LineSharedData) AddRecap(ctx smachine.ExecutionContext, ref reference.Global, rec *rms.RLineRecap) {
	p.ensureDataAccess()
	p.addSoloRecord(ctx, lineage.NewRecapRecord(catalog.Excerpt{
		RecordType:     uint32(rec.GetDefaultPolymorphID()),
		PrevRef:        rec.PrevRef,
		RootRef:        rec.RootRef,
	}, ref, rec))
	p.deps.ResolveDependency(ref)

	// 	if (rec.RedirectToType == 0) != rec.PrevRedirectRef.IsEmpty() {
	// 		panic(throw.IllegalValue())
	// 	}
	//
	// 	p.resolveDependency(ref, ResolvedDependency{
	// 		RecordType:     RecordType(rec.PrevType),
	// 		RootRef:        rec.RootRef.Get(),
	// 		RedirectToType: RecordType(rec.RedirectToType),
	// 		RedirectToRef:  rec.PrevRedirectRef.Get(),
	// 	})

}

//nolint
func (p *LineSharedData) addSoloRecord(ctx smachine.ExecutionContext, rec lineage.Record) (*buildersvc.Future, *buildersvc.Future) {
	p.ensureDataAccess()
	br := p.data.NewBundle()
	br.Add(rec)

	panic(throw.NotImplemented())
	// if prevFut, fut, _ := p.applyBundle(ctx, br); f != nil {
	// 	return prevFut, fut
	// }
	//
	// if len(br.GetUnresolvedDependencies()) == 0 {
	// 	panic(throw.Impossible())
	// }
	//
	// errs := br.GetErrors()
	// if len(errs) == 0 {
	// 	panic(throw.Impossible())
	// }
	// err := errs[0]
	// for _, e2 := range errs[1:] {
	// 	err = throw.WithDetails(err, e2)
	// }
	// panic(err)
}

func (p *LineSharedData) getUnresolved() UnresolvedDependencyMap {
	p.ensureAccess()
	return p.deps.GetPendingUnresolved()
}

func (p *LineSharedData) onDropReady(dd *DropSharedData) {
	p.dropFinalizeSync = dd.GetFinalizeSync()
}

func (p *LineSharedData) TrimStages() {
	p.ensureDataAccess()
	p.data.TrimCommittedStages()
}

func (p *LineSharedData) CollectSignatures(set inspectsvc.InspectedRecordSet, canBePartial bool) int {
	p.ensureDataAccess()

	for i := range set.Records {
		r := &set.Records[i]
		switch ok, _, sig := p.data.FindRegistrarSignature(r.RecRef); {
		case !ok:
			if canBePartial {
				return i
			}
			panic(throw.IllegalValue())
		case !sig.IsEmpty():
			r.RegistrarSignature = sig
		default:
			panic(throw.Impossible())
		}
	}

	return len(set.Records)
}

func (p *LineSharedData) FindWithTracker(ref reference.Holder) (bool, *buildersvc.Future, lineage.ReadRecord) {
	p.ensureDataAccess()

	switch ok, tracker, rec := p.data.FindWithTracker(ref); {
	case !ok:
		return false, nil, lineage.ReadRecord{}
	case tracker != nil:
		return true, tracker.(*buildersvc.Future), rec
	default:
		return true, nil, rec
	}
}

func (p *LineSharedData) FindSequence(selector dataextractor.Selector, findFn func(record lineage.ReadRecord) bool) (bool, *buildersvc.Future) {
	p.ensureDataAccess()

	p.data.RegisterRead()

	switch {
	case !selector.StartRef.IsEmpty():
		ok, fut := p.data.ScanFilamentFrom(selector.StartRef, selector.Direction.IsToPast(), findFn)
		if fut == nil {
			return ok, nil
		}
		return ok, fut.(*buildersvc.Future)

	case !selector.StopRef.IsEmpty():
		panic(throw.NotImplemented()) // TODO
		// ok, fut := p.data.ScanFilamentTill(selector.StopRef, selector.Direction.IsToPast(), findFn)
		// if fut == nil {
		// 	return ok, nil
		// }
		// return ok, fut.(*buildersvc.Future)

	case !selector.RootRef.IsEmpty():
		panic(throw.NotImplemented()) // TODO
		// return p.data.ScanFilamentAll(selector.RootRef, selector.Direction.IsToPast(), findFn), nil
	case !selector.ReasonRef.IsEmpty():
		panic(throw.Unsupported()) // ReasonRef must be resolved before this call
	default:
		panic(throw.IllegalValue())
	}
}

func (p *LineSharedData) DropID() jet.DropID {
	p.ensureDataAccess()
	return p.jetDropID
}
