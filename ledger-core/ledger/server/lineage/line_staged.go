// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ lineResolver = &LineStages{}

type LineStages struct {
	base  reference.Local
	pn    pulse.Number
	cache DependencyResolver

	earliest *updateStage
	latest   *updateStage

	recordRefs    map[reference.LocalHash]recordNo
	filamentRefs  map[reference.Local]filamentNo

	lineRecords
}

func (p *LineStages) NewBundle() *BundleResolver {
	return newBundleResolver(p, GetRecordPolicy)
}

func (p *LineStages) AddBundle(bundle *BundleResolver, tracker StageTracker) bool {
	switch {
	case tracker == nil:
		panic(throw.IllegalValue())
	case bundle == nil:
		panic(throw.IllegalValue())
	case bundle.IsEmpty():
		return true
	case !bundle.IsResolved():
		return false
	}

	prevFilamentCount := 0
	stage := &updateStage{ tracker: tracker }
	if p.latest != nil {
		stage.seqNo = p.latest.seqNo + 1
		f := p.latest.filaments
		prevFilamentCount = len(f)
		stage.filaments = append(make([]filament, 0, prevFilamentCount + 1), f...)
	} else {
		stage.seqNo = 1
		stage.filaments = make([]filament, 0, 2)
	}
	stage.firstRec = p.getNextRecNo()

	switch {
	case !bundle.isLineStart:
	case p.latest != nil:
		panic(throw.IllegalState())
	case bundle.records[0].filNo != 1:
		panic(throw.IllegalState())
	default:
		stage.filaments = append(stage.filaments, filament{})
	}

	filNo := filamentNo(0)
	var filRoot reference.Local

	firstRec := bundle.maxRecNo
	if firstRec > stage.firstRec {
		panic(throw.IllegalState())
	}

	recDelta := stage.firstRec - firstRec

	bundle.maxRecNo += recDelta

	defer bundle.setLastRecord(nil)

	for i := range bundle.records {
		rec := &bundle.records[i]

		bundle.setLastRecord(rec.GetRecordRef())

		rec.recordNo += recDelta

		if rec.recapNo >= firstRec {
			rec.recapNo += recDelta
		}

		isRecap := false
		switch {
		case rec.prev >= firstRec:
			rec.prev += recDelta
		case rec.prev > 0:
		case bundle.isLineStart && rec.recordNo == 1:
		case !bundle.isLineStart && RecordType(rec.Excerpt.RecordType) == tRSidelineStart: // TODO proper check for SidelineStart
		case RecordType(rec.Excerpt.RecordType) == tRLineRecap:
			if rec.recapNo != 0 {
				bundle.addError(throw.New("recap with recap"))
				continue
			}
			isRecap = true
		case rec.recapNo == 0:
			bundle.addError(throw.New("recap is missing"))
			continue
		case rec.recapNo >= rec.prev:
			bundle.addError(throw.New("recap in future"))
			continue
		}

		switch {
		case rec.next == deadFilament:
		case rec.next >= firstRec:
			rec.next += recDelta
		case rec.next > 0:
			bundle.addError(throw.New("next in past"))
			continue
		}

		switch {
		case rec.filNo != 0:
		case filNo != 0:
			rec.filNo = filNo
		case bundle.branchStart == 0 || bundle.branchHead == 0:
			bundle.addError(throw.New("inconsistent filament data"))
			continue
		case bundle.branchHead + recDelta != rec.recordNo:
			bundle.addError(throw.New("inconsistent filament head"))
			continue
		default:
			stage.filaments = append(stage.filaments, filament{})
			filNo = filamentNo(len(stage.filaments))
			rec.filNo = filNo
			filRoot = rec.GetRecordRef().GetLocal()
		}

		switch filament := &stage.filaments[rec.filNo - 1]; {
		case filament.earliest == 0:
			if isRecap {
				if filament.recap != 0 {
					panic(throw.Impossible())
				}
				filament.recap = rec.recordNo
				// TODO resolvedHead
				break
			}
			filament.earliest = rec.recordNo
			filament.latest = rec.recordNo
			filament.resolvedHead = rec.asResolvedDependency()
			if rec.next == deadFilament {
				filament.state = ended
			}

		case filament.latest != rec.prev:
			bundle.addError(throw.New("inconsistent filament sequence"))
			continue
		default:
			if rec.recapNo != 0 && filament.recap != rec.recapNo {
				bundle.addError(throw.New("inconsistent filament recap"))
				continue
			}
			filament.latest = rec.recordNo
			if rec.next == deadFilament {
				filament.state = ended
			}
		}
	}

	if filNo > 0 {
		// sanity check
		latest := stage.filaments[filNo - 1].latest
		switch bundle.records[latest - bundle.maxRecNo].next {
		case 0, deadFilament:
		default:
			bundle.addError(throw.New("inconsistent latest of filament"))
		}
	}

	if !bundle.IsResolved() {
		return false
	}

	if p.recordRefs == nil {
		p.recordRefs = map[reference.LocalHash]recordNo{}
	}

	if p.filamentRefs == nil {
		p.filamentRefs = map[reference.Local]filamentNo{}
	}

	for i := range bundle.records {
		rec := &bundle.records[i]

		if stage.firstRec + recordNo(i) != rec.recordNo {
			panic(throw.Impossible())
		}

		if prev := rec.prev; prev > 0 && prev < stage.firstRec {
			prevRec := p.get(prev)
			if prevRec.next == 0 && prevRec.filNo == rec.filNo {
				prevRec.next = rec.recordNo
			}
		}

		p.add(*rec)
		key := rec.GetRecordRef().GetLocal().IdentityHash()
		p.recordRefs[key] = rec.recordNo
	}

	if prevFilamentCount == 0 {
		p.filamentRefs[p.base] = 1
		prevFilamentCount++
	}

	if prevFilamentCount != len(stage.filaments) {
		p.filamentRefs[filRoot] = filamentNo(len(stage.filaments))
	}

	if p.latest != nil {
		p.latest.next = stage
	} else {
		p.earliest = stage
	}
	p.latest = stage

	// block this bundle from being reused without reprocessing
	bundle.addError(throw.New("discarded bundle"))

	return true
}

func (p *LineStages) TrimCommittedStages() {
	p.trimCommittedStages()
}

func (p *LineStages) trimCommittedStages() (last *updateStage) {
	for p.earliest != nil {
		switch tr := p.earliest.tracker; {
		case tr == nil:
			//
		case !tr.IsCommitted():
			return
		default:
			p.earliest.tracker = nil
		}
		last = p.earliest

		next := p.earliest.next
		if next == nil {
			return
		}
		p.earliest = next
	}
	return
}

func (p *LineStages) RollbackUncommittedRecords() {
	if p.earliest == nil {
		return
	}

	cutOffRec := deadFilament
	switch last := p.trimCommittedStages(); {
	case p.earliest.tracker == nil:
		// committed all
		p.latest = p.earliest
	case last == nil:
		// there is nothing committed
		*p = LineStages{
			base:         p.base,
			pn:           p.pn,
			cache:        p.cache,
		}
		return
	default:
		cutOffRec = p.earliest.firstRec
		p.lineRecords.truncate(cutOffRec)
		p.earliest = last
	}
	p.latest = p.earliest
	p.earliest.next = nil

	if cutOffRec == deadFilament {
		// everything was committed
		return
	}

	// cleanup the record map
	for k, recNo := range p.recordRefs {
		if recordNo(len(p.recordRefs)) < cutOffRec {
			break
		}
		if recNo >= cutOffRec {
			delete(p.recordRefs, k)
		}
	}

	// cleanup the filament map
	cutOffFil := filamentNo(len(p.latest.filaments) + 1)
	for k, filNo := range p.filamentRefs {
		if filamentNo(len(p.filamentRefs)) < cutOffFil {
			break
		}
		if filNo >= cutOffFil {
			delete(p.filamentRefs, k)
		}
	}

	// make open last records of filaments
	for _, f := range p.latest.filaments {
		latestRec := p.get(f.latest)
		if next := latestRec.next; next >= cutOffRec && next != deadFilament {
			latestRec.next = 0
		}
	}
}

func (p *LineStages) getNextFilNo() filamentNo {
	return filamentNo(len(p.filamentRefs)) + 1
}

func (p *LineStages) getLineBase() reference.LocalHolder {
	return p.base
}

func (p *LineStages) getLocalPN() pulse.Number {
	return p.pn
}

func (p *LineStages) findOtherDependency(ref reference.Holder) (ResolvedDependency, error) {
	if p.cache == nil || reference.IsEmpty(ref) {
		return ResolvedDependency{}, nil
	}
	return p.cache.FindOtherDependency(ref)
}

func (p *LineStages) findLineAnyDependency(root reference.Holder, ref reference.LocalHolder) (ResolvedDependency, error) {
	// TODO caching
	if p.cache == nil || ref == nil || ref.IsEmpty() {
		return ResolvedDependency{}, nil
	}
	return p.cache.FindLineAnyDependency(root, ref)
}

func (p *LineStages) findLineDependency(root reference.Holder, ref reference.LocalHolder, mustBeOpen bool) (filNo filamentNo, dep ResolvedDependency, recap recordNo) {
	// TODO caching
	return 0, ResolvedDependency{}, 0
}

func (p *LineStages) findLocalDependency(root reference.Holder, ref reference.LocalHolder, mustBeOpen bool) (filNo filamentNo, recNo recordNo, dep ResolvedDependency) {
	recNo = p.recordRefs[ref.GetLocal().IdentityHash()]
	if recNo == 0 {
		return 0, 0, ResolvedDependency{}
	}

	filNo = p.filamentRefs[root.GetLocal()]
	if filNo == 0 {
		return 0, 0, ResolvedDependency{}
	}

	rec := p.get(recNo)
	if mustBeOpen && rec.next != 0 {
		filNo = 0 // mark
	}

	return filNo, recNo, rec.asResolvedDependency()
}

func (p *LineStages) findFilament(root reference.LocalHolder) (filamentNo, ResolvedDependency) {
	filNo := p.filamentRefs[root.GetLocal()]
	if filNo == 0 {
		return 0, ResolvedDependency{}
	}

	return filNo, p.latest.filaments[filNo - 1].resolvedHead
}

func (p *LineStages) findCollision(local reference.LocalHolder, record *Record) (recordNo, error) {
	recNo := p.recordRefs[local.GetLocal().IdentityHash()]
	if recNo == 0 {
		return 0, nil
	}
	found := p.get(recNo)
	if found.Record.Equal(*record) {
		return recNo, nil
	}
	return 0, throw.E("record content mismatch", struct { Existing, New Record	}{ found.Record, *record })
}
