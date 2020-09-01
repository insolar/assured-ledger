// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ lineResolver = &LineStages{}

func NewStages(base reference.Local, pn pulse.Number, cache DependencyResolver) *LineStages {
	switch {
	case base.IsEmpty():
		panic(throw.IllegalValue())
	case base.SubScope() != reference.SubScopeLifeline:
		panic(throw.IllegalValue())
	default:
		switch basePN := base.GetPulseNumber(); {
		case !basePN.IsTimePulse():
			panic(throw.IllegalValue())
		case pn < basePN:
			panic(throw.IllegalValue())
		}
	}

	return &LineStages{
		base: base,
		pn: pn,
		cache: cache,
	}
}

type LineStages struct {
	base  reference.Local
	pn    pulse.Number
	cache DependencyResolver

	earliest *updateStage
	latest   *updateStage

	recordRefs    map[reference.LocalHash]recordNo
	filamentRefs  map[reference.Local]filamentNo
	reasonRefs    map[reference.Global]recordNo

	lineRecords
}

func (p *LineStages) NewBundle() *BundleResolver {
	return newBundleResolver(p, GetRecordPolicy)
}

func (p *LineStages) AddBundle(bundle *BundleResolver, tracker StageTracker) (bool, StageTracker, UpdateBundle) {
	switch {
	case tracker == nil:
		panic(throw.IllegalValue())
	case bundle == nil:
		panic(throw.IllegalValue())
	case !bundle.hasNoTroubles():
		return false, nil, UpdateBundle{}
	case len(bundle.records) > 0:
	case len(bundle.dupRecords) == 0:
		return true, nil, UpdateBundle{}
	default:
		// all records were deduplicated
		// we have to find the latest relevant tracker
		// so the caller can wait on it

		latestRec := bundle.dupRecords[0]
		for _, rn := range bundle.dupRecords[1:] {
			if latestRec < rn {
				latestRec = rn
			}
		}
		if s := p.findStage(latestRec); s != nil {
			return true, s.tracker, UpdateBundle{}
		}
		return true, nil, UpdateBundle{}
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

	validator := newValidator(bundle, stage)

	defer bundle.setLastRecord(nil)

	for i := range bundle.records {
		rec := &bundle.records[i]
		bundle.setLastRecord(rec.GetRecordRef())

		if err := p.checkReason(rec); err != nil {
			bundle.addError(err)
			continue
		}

		if err := validator.adjustNext(rec); err != nil {
			bundle.addError(err)
			continue
		}

		if err := validator.adjustPrevAndRecap(rec); err != nil {
			bundle.addError(err)
			continue
		}

		if err := validator.applyFilament(rec); err != nil {
			bundle.addError(err)
			continue
		}
	}

	if err := validator.postCheck(); err != nil {
		bundle.addError(err)
	}

	if !bundle.hasNoTroubles() {
		// has errors or unresolved dependencies
		return false, nil, UpdateBundle{}
	}

	// block this bundle from being reused without reprocessing - to protect bundle.records
	defer bundle.addError(throw.New("discarded bundle"))

	if p.addStage(bundle, stage, prevFilamentCount, validator.filRoot) {
		return true, tracker, UpdateBundle{bundle.records }
	}
	return false, nil, UpdateBundle{}
}

func (p *LineStages) addStage(bundle *BundleResolver, stage *updateStage, prevFilamentCount int, filRoot reference.Local) bool {
	cutOffRec := stage.firstRec

	defer func() {
		if cutOffRec > 0 {
			p.restoreLatest(cutOffRec)
		} else {
			p.restoreEmpty()
		}
	}()

	if p.recordRefs == nil {
		p.recordRefs = map[reference.LocalHash]recordNo{}
	}
	if p.filamentRefs == nil {
		p.filamentRefs = map[reference.Local]filamentNo{}
	}

	recNo := stage.firstRec
	for i := range bundle.records {
		rec := &bundle.records[i]

		if recNo != rec.recordNo {
			panic(throw.Impossible())
		}
		recNo++

		if prev := rec.prev; prev > 0 && prev < stage.firstRec {
			prevRec := p.get(prev)
			if prevRec.next == 0 && prevRec.filNo == rec.filNo {
				prevRec.next = rec.recordNo
			}
		}

		p.add(*rec)
		key := rec.GetRecordRef().GetLocal().IdentityHash()
		p.recordRefs[key] = rec.recordNo
		p.putReason(rec)
	}

	if recNo == stage.firstRec {
		panic(throw.Impossible())
	}

	if prevFilamentCount == 0 {
		p.filamentRefs[p.base] = 1
		prevFilamentCount++
	}

	if prevFilamentCount != len(stage.filaments) {
		p.filamentRefs[filRoot] = filamentNo(len(stage.filaments))
	}

	cutOffRec = deadFilament // disable cutoff by defer

	if p.latest != nil {
		p.latest.next = stage
	} else {
		p.earliest = stage
	}
	p.latest = stage

	return true
}

func (p *LineStages) findStage(recNo recordNo) *updateStage {
	var prev *updateStage

	for stage := p.earliest; stage != nil; prev, stage = stage, stage.next {
		switch {
		case stage.firstRec > recNo:
			return prev
		case stage.firstRec == recNo:
			return stage
		}
	}

	return nil
}

func (p *LineStages) TrimCommittedStages() {
	p.trimCommittedStages()
}

func (p *LineStages) trimCommittedStages() (last *updateStage) {
	for p.earliest != nil {
		next := p.earliest.next

		if tr := p.earliest.tracker; tr != nil {
			isReady, allocations := tr.GetFutureAllocation()
			if !isReady {
				return
			}
			p.earliest.tracker = nil
			if len(allocations) > 0 {
				p.setAllocations(p.earliest, allocations)
			} else {
				// TODO rollback and reapply
				panic(throw.NotImplemented())
			}
		}
		last = p.earliest

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
		p.restoreEmpty()
		return
	default:
		cutOffRec = p.earliest.firstRec
		p.earliest = last
	}
	p.latest = p.earliest
	p.earliest.next = nil

	p.restoreLatest(cutOffRec)
}

func (p *LineStages) RollbackLastBundle(tracker StageTracker) {
	switch {
	case tracker == nil:
		panic(throw.IllegalValue())
	case p.latest == nil:
		panic(throw.IllegalState())
	case p.latest.tracker != tracker:
		panic(throw.IllegalValue())
	}

	if p.latest == p.earliest {
		p.restoreEmpty()
		return
	}

	next := p.earliest.next
	for ; next != p.latest; next = next.next {}

	cutOffRec := p.latest.firstRec
	next.next = nil
	p.latest = next

	p.restoreLatest(cutOffRec)
}


func (p *LineStages) restoreLatest(cutOffRec recordNo) {
	if cutOffRec == deadFilament {
		// everything was committed
		return
	}

	// TODO use truncated records for cleanup
	p.lineRecords.truncate(cutOffRec)

	// cleanup reason map
	for k, recNo := range p.reasonRefs {
		if recNo >= cutOffRec {
			delete(p.reasonRefs, k)
		}
	}

	// cleanup record map
	for k, recNo := range p.recordRefs {
		if recordNo(len(p.recordRefs)) < cutOffRec {
			break
		}
		if recNo >= cutOffRec {
			delete(p.recordRefs, k)
		}
	}

	// cleanup filament map
	cutOffFil := filamentNo(len(p.latest.filaments) + 1)
	for k, filNo := range p.filamentRefs {
		if filamentNo(len(p.filamentRefs)) < cutOffFil {
			break
		}
		if filNo >= cutOffFil {
			delete(p.filamentRefs, k)
		}
	}

	// mark open last records of filaments
	for _, f := range p.latest.filaments {
		latestRec := p.get(f.latest)
		if next := latestRec.next; next >= cutOffRec && next != deadFilament {
			latestRec.next = 0
		}
	}
}

func (p *LineStages) restoreEmpty() {
	*p = LineStages{
		base:         p.base,
		pn:           p.pn,
		cache:        p.cache,
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

func (p *LineStages) findLineDependency(root reference.Holder, ref reference.LocalHolder) (ResolvedDependency, error) {
	// TODO caching
	if p.cache == nil || ref == nil || ref.IsEmpty() {
		return ResolvedDependency{}, nil
	}
	return p.cache.FindLineDependency(root, ref)
}

func (p *LineStages) findChainedDependency(root reference.Holder, ref reference.LocalHolder, mustBeOpen bool) (filNo filamentNo, recNo recordNo, dep ResolvedDependency, recap recordNo) {
	if ok, filNo, rec := p._findLocalDependency(root, ref); ok {
		if filNo == 0 {
			// this is filament mismatched or nil refs - avoid further search
			return 0, 0, ResolvedDependency{}, 0
		}
		recap := p.latest.filaments[filNo - 1].recap

		if mustBeOpen && rec.next != 0 {
			filNo = 0 // mark
		}
		return filNo, rec.recordNo, rec.asResolvedDependency(), recap
	}

	if p.cache != nil {
		// TODO implement
		// Open status should be checked by filaments
		// if mustBeOpen && rec.next != 0 {
		// 	filNo = 0 // mark
		// }
		_, _ = p.cache.FindLineDependency(root, ref)
		panic(throw.NotImplemented())
	}

	return 0, 0, ResolvedDependency{}, 0
}

func (p *LineStages) _findLocalDependency(root reference.LocalHolder, ref reference.LocalHolder) (bool, filamentNo, *updateRecord) {
	if root == nil || ref == nil || root.IsEmpty() || ref.IsEmpty() {
		return true, 0, nil
	}

	recNo := p.recordRefs[ref.GetLocal().IdentityHash()]
	if recNo == 0 {
		return false, 0, nil
	}
	rec := p.get(recNo)

	filNo := p.filamentRefs[root.GetLocal()]
	if filNo != rec.filNo {
		return true, 0, rec
	}

	return true, filNo, rec
}

func (p *LineStages) findLocalDependency(root reference.LocalHolder, ref reference.LocalHolder) (filamentNo, recordNo, ResolvedDependency) {
	if _, filNo, rec := p._findLocalDependency(root, ref); filNo != 0 {
		return filNo, rec.recordNo, rec.asResolvedDependency()
	}
	return 0, 0, ResolvedDependency{}
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
	if found.Record.EqualForRecordIdempotency(*record) {
		return recNo, nil
	}
	return 0, throw.E("record content mismatch", struct { Existing, New Record	}{ found.Record, *record })
}

func (p *LineStages) checkReason(rec *resolvedRecord) error {
	reasonRef := rec.Excerpt.ReasonRef.Get()
	if reference.IsEmpty(reasonRef) {
		return nil
	}
	normRef := reference.NormCopy(reasonRef)
	if _, ok := p.reasonRefs[normRef]; !ok {
		return nil
	}
	return throw.New("reused reason", struct { ReasonRef reference.Global }{ normRef })
}

func (p *LineStages) putReason(rec *resolvedRecord) {
	reasonRef := rec.Excerpt.ReasonRef.Get()
	if reference.IsEmpty(reasonRef) {
		return
	}
	normRef := reference.NormCopy(reasonRef)
	if p.reasonRefs == nil {
		p.reasonRefs = map[reference.Global]recordNo{}
	}
	p.reasonRefs[normRef] = rec.recordNo
}

func (p *LineStages) setAllocations(stage *updateStage, allocBase []ledger.DirectoryIndex) {
	max := uint32(0)
	if stage.next == nil {
		max = uint32(p.getNextRecNo() - stage.firstRec)
	} else {
		max = uint32(stage.next.firstRec - stage.firstRec)
	}

	if max != uint32(len(allocBase)) {
		panic(throw.IllegalState())
	}

	for i := uint32(0); i < max; i++ {
		rec := p.get(stage.firstRec + recordNo(i))
		rec.storageIndex = allocBase[i]
		rec.cleanup()
	}
}

func (p *LineStages) Find(ref reference.Holder) (found bool, recordIndex ledger.DirectoryIndex, recordInfo Record) {
	if rn, ok := p.recordRefs[ref.GetLocal().IdentityHash()]; ok {
		if r := p.get(rn); r != nil {
			return true, r.storageIndex, r.Record
		}
	}
	return false, 0, Record{}
}

func (p *LineStages) FindWithTracker(ref reference.Holder) (found bool, recordIndex ledger.DirectoryIndex, tracker StageTracker, recordInfo Record) {
	if rn, ok := p.recordRefs[ref.GetLocal().IdentityHash()]; ok {
		if r := p.get(rn); r != nil {
			if stage := p.findStage(rn); stage != nil {
				tracker = stage.tracker
			}
			return true, r.storageIndex, tracker, r.Record
		}
	}
	return false, 0, nil, Record{}
}


