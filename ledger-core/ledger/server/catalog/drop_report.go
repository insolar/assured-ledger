package catalog

import (
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type DropReport struct {
	ReportRec *rms.RCtlDropReport
}

func (v DropReport) IsZero() bool {
	return v.ReportRec == nil
}

func (v DropReport) Equal(other DropReport) bool {
	switch {
	case v.ReportRec == other.ReportRec:
		return true
	case v.ReportRec == nil || other.ReportRec == nil:
		return false
	default:
		return v.ReportRec.Equal(*other.ReportRec)
	}
}

