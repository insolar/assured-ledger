// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/ledger/jet"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/catalog"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/inspectsvc"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/lineage"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type LineKey reference.Global

type LineSharedData struct {
	lineRef reference.Global
	limiter smsync.SemaphoreLink
	ready   bool
	valid   bool

	jetDropID jet.DropID
	resolver  lineage.DependencyResolver
	adapter   buildersvc.Adapter

	data *lineage.LineStages
	deps DependencyTracker
}

func (p *LineSharedData) LineRef() reference.Global {
	return p.lineRef
}

func (p *LineSharedData) GetLimiter() smachine.SyncLink {
	return p.limiter.SyncLink()
}

func (p *LineSharedData) ensureDataAccess() {
	p.ensureAccess()
	if !p.jetDropID.IsValid() {
		panic(throw.IllegalState())
	}
	if p.data == nil {
		p.data = lineage.NewStages(p.lineRef.GetBase(), p.jetDropID.CreatedAt(), p.resolver)
	}
}

func (p *LineSharedData) TryApplyRecordSet(ctx smachine.ExecutionContext, set inspectsvc.InspectedRecordSet) (*buildersvc.Future, *lineage.BundleResolver) {
	if set.IsEmpty() {
		panic(throw.IllegalValue())
	}
	p.ensureDataAccess()

	br := p.data.NewBundle()
	br.Hint(set.Count())

	for _, r := range set.Records {
		br.Add(r)
	}

	return p.applyBundle(ctx, br)
}

func (p *LineSharedData) applyBundle(ctx smachine.ExecutionContext, br *lineage.BundleResolver) (*buildersvc.Future, *lineage.BundleResolver) {
	if !br.IsResolved() {
		return nil, br
	}

	future := buildersvc.NewFuture("")
	if !p.data.AddBundle(br, future) {
		return nil, br
	}

	dropID := p.jetDropID
	p.adapter.PrepareNotify(ctx, func(service buildersvc.Service) {
		service.AppendToDrop(dropID, future, br.GetResolved())
	}).Send()

	return future, nil
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

func (p *LineSharedData) enableAccess() smachine.SyncAdjustment {
	p.ready = true
	return p.limiter.NewValue(1)
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

func (p *LineSharedData) addSoloRecord(ctx smachine.ExecutionContext, rec lineage.Record) *buildersvc.Future {
	p.ensureDataAccess()
	br := p.data.NewBundle()
	br.Add(rec)

	if br.IsEmpty() {
		// was deduplicated
		return nil
	}

	if f, _ := p.applyBundle(ctx, br); f != nil {
		return f
	}

	if len(br.GetUnresolvedDependencies()) == 0 {
		panic(throw.Impossible())
	}

	errs := br.GetErrors()
	if len(errs) == 0 {
		panic(throw.Impossible())
	}
	err := errs[0]
	for _, e2 := range errs[1:] {
		err = throw.WithDetails(err, e2)
	}
	panic(err)
}

func (p *LineSharedData) getUnresolved() UnresolvedDependencyMap {
	p.ensureAccess()
	return p.deps.GetPendingUnresolved()
}

func (p *LineSharedData) onDropReady(sd *DropSharedData) {

}
