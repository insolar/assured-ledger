// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

type resolvedRecord struct {
	prev recordNo
	next recordNo

	recordNo recordNo
	stageNo  stageNo
	filNo    filamentNo
	recapNo  recordNo

	redirectToType RecordType
//	policy RecordPolicy
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
	if redirectToRef := p.Excerpt.RedirectRef.Get(); !redirectToRef.IsEmpty() {
		rd.RedirectToRef = redirectToRef
		rd.RedirectToType = p.redirectToType
	}
	return rd
}
