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

func (p *stageValidator) _addFilament(rec *resolvedRecord) error {
	switch {
	case rec.filNo != 0:
	case p.filNo != 0:
		rec.filNo = p.filNo
	case p.bundle.branchRoot == 0 || p.bundle.branchFirst == 0:
		return throw.New("inconsistent filament data")
	case p.bundle.branchFirst + p.recDelta != rec.recordNo:
		return throw.New("inconsistent filament head")
	default:
		p.stage.filaments = append(p.stage.filaments, filament{})
		p.filNo = filamentNo(len(p.stage.filaments))
		rec.filNo = p.filNo
		p.filRoot = rec.GetRecordRef().GetLocal()
	}
	return nil
}

func (p *stageValidator) applyFilament(rec *resolvedRecord) (recordNo, error) {
	if err := p._addFilament(rec); err != nil {
		return 0, err
	}

	filament := &p.stage.filaments[rec.filNo - 1]

	switch {
	case filament.earliest == 0:
		//
	case filament.latest != rec.prev:
		return 0, throw.New("inconsistent filament sequence")
	default:
		if rec.recapNo != 0 && filament.recap != rec.recapNo {
			return 0, throw.New("inconsistent filament recap")
		}
		filament.setLatest(rec.recordNo)
		return filament.root, nil
	}

	if rec.filNo != 1 {
		switch {
		case p.bundle.branchRoot == 0:
			return 0, throw.New("inconsistent bundle branching root")
		case p.bundle.branchFirst == 0:
			return 0, throw.New("inconsistent bundle branching")
		case p.bundle.branchFirst + p.recDelta != rec.recordNo:
			return 0, throw.New("unexpected filament record")
		default:
			filament.setRoot(p.bundle.branchRoot + p.recDelta)
		}
	}

	if !rec.IsRecap() {
		filament.resolvedHead = rec.asResolvedDependency()
		filament.setLatest(rec.recordNo)
		return filament.root, nil
	}

	switch {
	case filament.recap != 0:
		panic(throw.Impossible())
	case rec.recapRec == nil:
		return 0, throw.New("recap body is missing")
	case rec.recapRec.State & rms.FilamentState_FilamentTypeMask == rms.FilamentState_Lifeline:
		if rec.filNo != 1 {
			return 0, throw.New("recap mismatched line type")
		}
	case rec.filNo == 1:
		return 0, throw.New("recap mismatched filament type")
	}

	switch {
	case rec.recapRec.State & rms.FilamentState_FilamentTypeMask == rms.FilamentState_Unknown:
		return 0, throw.New("recap is invalid")
	case rec.recapRec.State & rms.FilamentState_ClosedFlag != 0:
		filament.setLatest(deadFilament)
	}

	filament.recap = rec.recordNo
	filament.resolvedHead = ResolvedDependency{
		RecordType:     RecordType(rec.recapRec.PrevType),
		RootRef:        rec.Excerpt.RootRef.Get(),
	}

	if rType := RecordType(rec.recapRec.RedirectToType); rType != 0 {
		filament.resolvedHead.RedirectToRef = rec.recapRec.PrevRedirectRef.Get()
		if reference.IsEmpty(filament.resolvedHead.RedirectToRef) {
			return 0, throw.New("recap inconsistent redirect")
		}
		filament.resolvedHead.RedirectToType = RecordType(rec.recapRec.RedirectToType)
	}

	return filament.root, nil
}

func (p *stageValidator) postCheck() error {
	if p.filNo == 0 {
		return nil
	}

	// sanity check
	if latest := p.stage.filaments[p.filNo - 1].latest; latest != deadFilament {
		// filament was not received as closed

		switch p.bundle.records[latest - p.bundle.maxRecNo].next {
		case 0, deadFilament:
		default:
			return throw.New("inconsistent latest of filament")
		}
	}

	if p.filRoot.IsEmpty() {
		return throw.New("illegal filament root")
	}

	return nil
}

