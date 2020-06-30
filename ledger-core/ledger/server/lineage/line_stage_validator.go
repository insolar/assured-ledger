// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func newValidator(bundle *BundleResolver, stage *updateStage) *stageValidator {
	stageAdd := &stageValidator{ bundle: bundle, stage: stage, firstRec: bundle.maxRecNo }

	if stageAdd.firstRec > stage.firstRec {
		panic(throw.IllegalState())
	}
	stageAdd.recDelta = stage.firstRec - stageAdd.firstRec
	bundle.maxRecNo += stageAdd.recDelta

	return stageAdd
}

type stageValidator struct {
	bundle   *BundleResolver
	stage    *updateStage
	filNo    filamentNo
	firstRec recordNo
	recDelta recordNo
	filRoot  reference.Local
}

func (p *stageValidator) adjustNext(rec *resolvedRecord) error {
	rec.recordNo += p.recDelta

	switch {
	case rec.next == deadFilament:
	case rec.next >= p.firstRec:
		rec.next += p.recDelta
	case rec.next > 0:
		return throw.New("next in past")
	}
	return nil
}

func (p *stageValidator) adjustPrevAndRecap(rec *resolvedRecord) error {
	if rec.recapNo >= p.firstRec {
		rec.recapNo += p.recDelta
	}

	switch {
	case rec.prev >= p.firstRec:
		rec.prev += p.recDelta
	case rec.prev > 0:
	case p.bundle.isLineStart && rec.recordNo == 1:
	case !p.bundle.isLineStart && rec.IsSidelineStart():
	case rec.IsRecap():
		if rec.recapNo != 0 {
			return throw.New("recap with recap")
		}
	case rec.recapNo == 0:
		return throw.New("recap is missing")
	case rec.recapNo >= rec.prev:
		return throw.New("recap in future")
	}

	return nil
}

func (p *stageValidator) addFilament(rec *resolvedRecord) error {
	switch {
	case rec.filNo != 0:
	case p.filNo != 0:
		rec.filNo = p.filNo
	case p.bundle.branchStart == 0 || p.bundle.branchHead == 0:
		return throw.New("inconsistent filament data")
	case p.bundle.branchHead + p.recDelta != rec.recordNo:
		return throw.New("inconsistent filament head")
	default:
		p.stage.filaments = append(p.stage.filaments, filament{})
		p.filNo = filamentNo(len(p.stage.filaments))
		rec.filNo = p.filNo
		p.filRoot = rec.GetRecordRef().GetLocal()
	}
	return nil
}

func (p *stageValidator) applyFilament(rec *resolvedRecord) error {
	filament := &p.stage.filaments[rec.filNo - 1]

	switch {
	case filament.earliest == 0:
		//
	case filament.latest != rec.prev:
		return throw.New("inconsistent filament sequence")
	default:
		if rec.recapNo != 0 && filament.recap != rec.recapNo {
			return throw.New("inconsistent filament recap")
		}
		filament.latest = rec.recordNo
		if rec.next == deadFilament {
			filament.state = ended
		}
		return nil
	}

	if !rec.IsRecap() {
		filament.earliest = rec.recordNo
		filament.latest = rec.recordNo
		filament.resolvedHead = rec.asResolvedDependency()
		if rec.next == deadFilament {
			filament.state = ended
		}

		return nil
	}

	switch {
	case filament.recap != 0:
		panic(throw.Impossible())
	case rec.RecapRec == nil:
		return throw.New("recap body is missing")
	case rec.RecapRec.Type == rms.FilamentType_Lifeline:
		if rec.filNo != 1 {
			return throw.New("recap mismatched line type")
		}
	case rec.filNo == 1:
		return throw.New("recap mismatched filament type")
	}

	switch rec.RecapRec.State {
	case rms.FilamentState_Unknown:
		return throw.New("recap is invalid")
	case rms.FilamentState_DirtyDeactivated, rms.FilamentState_Dead:
		filament.earliest = deadFilament
		filament.latest = deadFilament
	}

	filament.recap = rec.recordNo
	filament.resolvedHead = ResolvedDependency{
		RecordType:     RecordType(rec.RecapRec.PrevType),
		RootRef:        rec.Excerpt.RootRef.Get(),
	}

	if rType := RecordType(rec.RecapRec.RedirectToType); rType != 0 {
		filament.resolvedHead.RedirectToRef = rec.RecapRec.RedirectRef.Get()
		if reference.IsEmpty(filament.resolvedHead.RedirectToRef) {
			return throw.New("recap inconsistent redirect")
		}
		filament.resolvedHead.RedirectToType = RecordType(rec.RecapRec.RedirectToType)
	}

	return nil
}

func (p *stageValidator) postCheck() error {
	if p.filNo == 0 {
		return nil
	}

	// sanity check
	latest := p.stage.filaments[p.filNo - 1].latest
	switch p.bundle.records[latest - p.bundle.maxRecNo].next {
	case 0, deadFilament:
	default:
		return throw.New("inconsistent latest of filament")
	}

	if p.filRoot.IsEmpty() {
		return throw.New("illegal filament root")
	}

	return nil
}

