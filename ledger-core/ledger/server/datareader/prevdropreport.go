// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package datareader

import (
	"github.com/insolar/assured-ledger/ledger-core/rms"
<<<<<<< HEAD
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
=======
>>>>>>> SMs
)

type PrevDropReport struct {
	ReportRec *rms.RPrevDropReport
}

<<<<<<< HEAD
func (v PrevDropReport) IsZero() bool {
	return v.ReportRec == nil
}

func (v PrevDropReport) Equal(other PrevDropReport) bool {
	panic(throw.NotImplemented())
=======
func (p PrevDropReport) Equal(other PrevDropReport) bool {

>>>>>>> SMs
}

