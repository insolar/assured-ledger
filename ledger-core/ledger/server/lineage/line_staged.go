// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"errors"

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

	// block this bundle from being reused without reprocessing
	bundle.addError(errors.New("discarded bundle"))
	bundle.maxRecNo += recDelta

	for i := range bundle.records {
		rec := &bundle.records[i]
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
				panic(throw.IllegalState())
			}
			isRecap = true
		case rec.recapNo == 0:
			// TODO recap is expected error
			panic(throw.IllegalState())
		case rec.recapNo >= rec.prev:
			panic(throw.IllegalState())
		}

		switch {
		case rec.next >= firstRec:
			rec.next += recDelta
		case rec.next > 0:
			panic(throw.IllegalState())
		}

		switch {
		case rec.filNo != 0:
		case filNo != 0:
			rec.filNo = filNo
		case bundle.branchStart == 0 || bundle.branchHead == 0:
			panic(throw.IllegalState())
		case bundle.branchHead + recDelta != rec.recordNo:
			panic(throw.IllegalState())
		default:
			stage.filaments = append(stage.filaments, filament{})
			filNo = filamentNo(len(stage.filaments))
			rec.filNo = filNo
			filRoot = rec.RegRecord.AnticipatedRef.Get().GetLocal()
		}

		switch filament := &stage.filaments[rec.filNo - 1]; {
		case filament.earliest == 0:
			if isRecap {
				if filament.recap != 0 {
					panic(throw.IllegalState())
				}
				filament.recap = rec.recordNo
				// TODO resolvedHead
				break
			}
			filament.earliest = rec.recordNo
			filament.latest = rec.recordNo
			filament.resolvedHead = rec.asResolvedDependency()

		case filament.latest != rec.prev:
			panic(throw.IllegalState())
		default:
			filament.latest = rec.recordNo
			if rec.recapNo != 0 && filament.recap != rec.recapNo {
				panic(throw.IllegalState())
			}
		}
	}

	if filNo > 0 {
		// sanity check
		latest := stage.filaments[filNo - 1].latest
		if bundle.records[latest - bundle.maxRecNo].next != 0 {
			panic(throw.IllegalState())
		}
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

		p.add(*rec)
		key := rec.RegRecord.AnticipatedRef.Get().GetLocal().IdentityHash()
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
	last := p.trimCommittedStages()
	if p.earliest.tracker == nil {
		// committed all
		p.latest = p.earliest
	} else {
		p.lineRecords.truncate(p.earliest.firstRec)
		p.earliest = last
	}
	p.latest = p.earliest
	p.earliest.next = nil
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

func (p *LineStages) findLineAnyDependency(root reference.Holder, ref reference.LocalHolder) ResolvedDependency {
	// TODO caching
	return ResolvedDependency{}
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
