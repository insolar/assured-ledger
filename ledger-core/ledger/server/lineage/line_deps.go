// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

func NewDependencyTracker() *DependencyTracker {
	return &DependencyTracker{}
}

type DependencyTracker struct {
}

func (p *DependencyTracker) AddDependencySet(deps []UnresolvedDependency, callbackFn func()) {

}

func (p *DependencyTracker) AddRecap(ref reference.Global, rec *rms.RLineRecap) {
	p.addRecordInfo(ref, ResolvedDependency{
		RecordType:     RecordType(rec.PrevType),
		RootRef:        rec.RootRef.Get(),
		RedirectToType: RecordType(rec.RedirectToType),
		RedirectToRef:  rec.PrevRedirectRef.Get(),
	})
}

func (p *DependencyTracker) AddRecordInfo(ref reference.Global, rec ResolvedDependency) {

}

func (p *DependencyTracker) addRecordInfo(ref reference.Global, rec ResolvedDependency) {

}
