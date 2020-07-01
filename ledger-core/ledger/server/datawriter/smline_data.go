// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datawriter

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
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

	jetDropID   JetDropID
	dropUpdater buildersvc.JetDropAssistant

	data *lineage.LineStages
	deps *lineage.DependencyTracker
	// recapRef    reference.Global
}

func (p *LineSharedData) LineRef() reference.Global {
	return p.lineRef
}

func (p *LineSharedData) GetLimiter() smachine.SyncLink {
	return p.limiter.SyncLink()
}

func (p *LineSharedData) ensureDataAccess() {
	p.ensureAccess()
	if p.dropUpdater == nil {
		panic(throw.IllegalState())
	}
	if p.data == nil {
		p.data = lineage.NewStages(p.lineRef.GetBase(), p.jetDropID.GetPulseNumber(), nil)
	}
}

func (p *LineSharedData) TryApplyRecordSet(set inspectsvc.InspectedRecordSet) (*buildersvc.Future, *lineage.BundleResolver) {
	if set.IsEmpty() {
		panic(throw.IllegalValue())
	}
	p.ensureDataAccess()

	br := p.data.NewBundle()
	br.Hint()
	// br.Add()

	if !br.IsResolved() {
		return nil, br
	}

	future := &buildersvc.Future{}
	if !p.data.AddBundle(br, future) {
		return nil, br
	}
	if !p.dropUpdater.AddRecords(future, br) {
		p.data.RollbackLastBundle(future)
		return nil, br
	}
	return future, nil
}

func (p *LineSharedData) RequestDependencies(br *lineage.BundleResolver, wakeup smachine.BargeInNoArgHolder) {
	p.ensureAccess()
	switch {
	case p.data == nil:
		panic(throw.IllegalState())
	case p.deps == nil:
		p.deps = lineage.NewDependencyTracker()
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

func (p *LineSharedData) addRecap(rec *rms.RLineRecap, ref reference.Global) {
	p.ensureAccess()
	if p.deps == nil {
		panic(throw.IllegalState())
	}
	p.deps.AddRecap(ref, rec)
}

func (p *LineSharedData) getMissingFilaments() []reference.Global {
	p.ensureAccess()
	if p.deps == nil {
		panic(throw.IllegalState())
	}
	panic(throw.NotImplemented())
}
