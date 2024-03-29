package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type resolvedRecord struct {
	prev recordNo
	next recordNo

	recordNo recordNo
	filNo    filamentNo
	recapNo  recordNo

	storageIndex       ledger.DirectoryIndex
	filamentStartIndex ledger.DirectoryIndexAndFlags

	redirectToType RecordType
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
	if redirectToRef := p.Excerpt.RedirectRef.Get(); !reference.IsEmpty(redirectToRef) {
		rd.RedirectToRef = redirectToRef
		rd.RedirectToType = p.redirectToType
	}
	return rd
}

func (p *resolvedRecord) IsRecap() bool {
	return RecordType(p.Excerpt.RecordType) == tRLineRecap // TODO proper check for LineRecap
}

func (p *resolvedRecord) IsSidelineStart() bool {
	return RecordType(p.Excerpt.RecordType) == tRSidelineStart // TODO proper check for SidelineStart
}
