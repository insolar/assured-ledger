package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type LineSummary struct {
	LineRecap FilamentSummary
	LineReport rms.RStateReport
	Filaments []FilamentSummary
}

func (v LineSummary) IsZero() bool {
	return v.LineRecap.IsZero()
}

type FilamentSummary struct {
	Local reference.Local
	Recap rms.RLineRecap
}

func (v FilamentSummary) IsZero() bool {
	return v.Local.IsEmpty()
}

