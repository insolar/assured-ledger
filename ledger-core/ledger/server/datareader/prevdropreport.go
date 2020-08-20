// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datareader

import (
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type PrevDropReport struct {
	ReportRec *rms.RPrevDropReport
}

func (v PrevDropReport) IsZero() bool {
	return v.ReportRec == nil
}

func (v PrevDropReport) Equal(other PrevDropReport) bool {
	switch {
	case v.ReportRec == other.ReportRec:
		return true
	case v.ReportRec == nil || other.ReportRec == nil:
		return false
	default:
		return v.ReportRec.Equal(*other.ReportRec)
	}
}

