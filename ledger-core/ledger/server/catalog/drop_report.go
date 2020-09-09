// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package catalog

import (
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type DropReport struct {
	ReportRec *rms.RPrevDropReport
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

