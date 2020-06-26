// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger/server/buildersvc"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type StagedLine struct {
	base reference.Local
	pn   pulse.Number

	earliest *updateStage
	latest   *updateStage

	recordRefs    map[reference.Local]recordNo
	filamentRefs  map[reference.Local]filamentNo

	records    [][]updateRecord
}

func (p *StagedLine) addRecord(record Record) error {
	ref := record.RegRecord.AnticipatedRef.Get()
	if err := p.checkBase(ref.GetBase(), true); err != nil {
		return err
	}

	localRef := ref.GetLocal()
	if err := p.checkLocal(localRef, true); err != nil {
		return err
	}

	if dupIdx := p.recordRefs[localRef]; dupIdx != 0 {
		rec := p.getRecord(dupIdx)
		if !rec.Record.Equal(record) {
			return throw.E("reference collision", struct { Ref reference.Global }{ reference.Copy(ref) })
		}
		return nil
	}

	upd := updateRecord{
		stageNo: p.latest.seqNo,
		Record: record,
	}
	upd.recordNo = p.getNextRecordNo()

	// policy := GetRecordPolicy(uint64(upd.Excerpt.RecordType))


	if upd.Excerpt.RootRef.IsZero() {
		// lifeline start
	} else {

	}

	if upd.Excerpt.PrevRef.IsZero() {
		// lifeline / sideline start
	} else {
		// check if recap is needed for prev

		var err error
		upd.prev, err = p.resolveRef(upd.Excerpt.PrevRef.Get())
		if err != nil {
			return err
		}

		prevRec := p.getRecord(upd.prev)
		if prevRec.next != 0 {
			return throw.E("branching detected", struct { Ref reference.Global }{ reference.Copy(ref) })
		}
		prevRec.next = upd.recordNo
	}

	p0 := p.records[0]
	if n := cap(p0); n == len(p0) {
		p.records = append(p.records, p0)
		p0 = make([]updateRecord, 1, n)
		p0[0] = upd
		p.records[0] = p0
	} else {
		p.records[0] = append(p0, upd)
	}

	p.recordRefs[localRef] = upd.recordNo

	// p.resolveRef(upd.Excerpt.PrevRef)
	// p.resolveRef(upd.Excerpt.RootRef)
	// p.resolveRef(upd.Excerpt.ReasonRef)
	// p.resolveRef(upd.Excerpt.RedirectRef)
}

func (p *StagedLine) addStage(future *buildersvc.Future) {
	stage := &updateStage{ future: future }
	if p.latest != nil {
		stage.seqNo = p.latest.seqNo + 1
		stage.filaments = p.latest.filaments
		p.latest.next = stage
	} else {
		p.earliest = stage
	}
	p.latest = stage
}

func (p *StagedLine) trimStages() {
	for p.earliest != p.latest {
		if !p.earliest.future.IsCommitted() {
			return
		}
		p.earliest = p.earliest.next
	}
}

func (p *StagedLine) checkBase(base reference.Local, exactPN bool) error {
	if base != p.base {
		return throw.E("wrong base", struct{ Expected, Actual reference.Local }{p.base, base})
	}
	return nil
}


func (p *StagedLine) checkLocal(local reference.Local, exactPN bool) error {
	if ss := local.SubScope(); ss != reference.SubScopeLifeline {
		return throw.E("invalid scope", struct { Actual reference.SubScope }{ ss })
	}
	switch pn := local.GetPulseNumber(); {
	case exactPN:
		if pn != p.pn {
			return throw.E("wrong pulse number", struct { Expected, Actual pulse.Number }{ p.pn, pn })
		}
	case pn >= p.pn:
		return throw.E("wrong past pulse number", struct { Expected, Actual pulse.Number }{ p.pn, pn })
	}
	return nil
}

func (p *StagedLine) getNextRecordNo() recordNo {
	return recordNo((len(p.records) - 1) * cap(p.records[0]) + len(p.records[0]) + 1)
}

func (p *StagedLine) getRecord(recNo recordNo) *updateRecord {
	n := len(p.records)
	switch {
	case recNo == 0:
		panic(throw.IllegalValue())
	case n <= 1:
		return &p.records[0][recNo - 1]
	}

	recNo--
	pgSize := cap(p.records[0])
	pg := 1 + int(recNo) / pgSize
	idx := int(recNo) % pgSize

	if pg == n {
		return &p.records[0][idx]
	}
	return &p.records[pg][idx]
}

func (p *StagedLine) resolveRef(ref reference.Holder) (recordNo, error) {

}
