// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

<<<<<<< HEAD
import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

=======
>>>>>>> Further work
type resolvedRecord struct {
	prev recordNo
	next recordNo

	recordNo recordNo
<<<<<<< HEAD
	filNo    filamentNo
	recapNo  recordNo

	storageIndex uint32

	redirectToType RecordType
=======
	stageNo  stageNo
	filNo    filamentNo
	recapNo  recordNo

	storageIndex uint32

	redirectToType RecordType
//	policy RecordPolicy
>>>>>>> Further work
	Record
}

func (p *resolvedRecord) asResolvedDependency() ResolvedDependency {
	if p == nil {
		return ResolvedDependency{}
	}

	rd := ResolvedDependency{
		RecordType: RecordType(p.Excerpt.RecordType),
		RootRef:    p.Excerpt.RootRef.Get(),
	}
<<<<<<< HEAD
	if redirectToRef := p.Excerpt.RedirectRef.Get(); !reference.IsEmpty(redirectToRef) {
=======
	if redirectToRef := p.Excerpt.RedirectRef.Get(); !redirectToRef.IsEmpty() {
>>>>>>> Further work
		rd.RedirectToRef = redirectToRef
		rd.RedirectToType = p.redirectToType
	}
	return rd
}
<<<<<<< HEAD

func (p *resolvedRecord) IsRecap() bool {
	return RecordType(p.Excerpt.RecordType) == tRLineRecap // TODO proper check for LineRecap
}

func (p *resolvedRecord) IsSidelineStart() bool {
	return RecordType(p.Excerpt.RecordType) == tRSidelineStart // TODO proper check for SidelineStart
}
=======
>>>>>>> Further work
