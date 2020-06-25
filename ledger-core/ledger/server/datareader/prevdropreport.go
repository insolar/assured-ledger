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

func (p PrevDropReport) Equal(other PrevDropReport) bool {

}

