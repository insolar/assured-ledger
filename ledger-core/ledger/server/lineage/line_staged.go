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

type LineAges struct {
	base reference.Local
	pn   pulse.Number

	earliest *updateStage
	latest   *updateStage

	recordRefs    map[reference.Local]recordNo
	filamentRefs  map[reference.Local]filamentNo

	records    [][]updateRecord
}

func (p *LineAges) NewBundle() *BundleResolver {
	return newBundleResolver(nil, GetRecordPolicy)
}

func (p *LineAges) trimStages() {
	for ;p.earliest != p.latest;p.earliest = p.earliest.next {
		if !p.earliest.future.IsCommitted() {
			return
		}
	}
}

func (p *LineAges) getNextRecordNo() recordNo {
	return recordNo((len(p.records) - 1) * cap(p.records[0]) + len(p.records[0]) + 1)
}

func (p *LineAges) getRecord(recNo recordNo) *updateRecord {
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
