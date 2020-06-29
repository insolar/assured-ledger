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
	base reference.Local
	pn   pulse.Number

	earliest *updateStage
	latest   *updateStage

	recordRefs    map[reference.LocalHash]recordNo
	filamentRefs  map[reference.Local]filamentNo

	lineRecords
}

func (p *LineStages) NewBundle() *BundleResolver {
	return newBundleResolver(nil, GetRecordPolicy)
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

func (p *LineStages) findLineAnyDependency(root reference.Holder, ref reference.LocalHolder) ResolvedDependency {
	// TODO caching
	panic("implement me")
}

func (p *LineStages) findLineDependency(root reference.Holder, ref reference.LocalHolder, mustBeOpen bool) (filNo filamentNo, dep ResolvedDependency, recap recordNo) {
	// TODO caching
	panic("implement me")
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
	if rec.filNo != filNo {

	}


	panic("implement me")
}

func (p *LineStages) findFilament(root reference.LocalHolder) (filamentNo, ResolvedDependency) {
	filNo := p.filamentRefs[root.GetLocal()]
	if filNo == 0 {
		return 0, ResolvedDependency{}
	}


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

// func (p *LineStages) trimStages() {
// 	for ;p.earliest != p.latest;p.earliest = p.earliest.next {
// 		if !p.earliest.future.IsCommitted() {
// 			return
// 		}
// 	}
// }
