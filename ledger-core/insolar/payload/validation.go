// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type Validatable interface {
	Validate(currPulse PulseNumber) error
}

func isTimePulseBefore(pn pulse.Number, before pulse.Number) bool {
	return pn.IsTimePulse() && pn.IsBefore(before)
}

func isSpecialOrTimePulseBefore(pn pulse.Number, before pulse.Number) bool {
	return pn.IsSpecial() || (pn.IsTimePulse() && pn.IsBefore(before))
}

func isTimePulseBeforeOrEq(pn pulse.Number, before pulse.Number) bool {
	return pn.IsTimePulse() && pn.IsBeforeOrEq(before)
}

func isSpecialOrTimePulseBeforeOrEq(pn pulse.Number, before pulse.Number) bool {
	return pn.IsSpecial() || (pn.IsTimePulse() && pn.IsBeforeOrEq(before))
}

func globalBasePulseIsSpecialOrBeforeOrEqLocalPulse(global reference.Global) bool {
	var (
		basePulse  = global.GetBase().GetPulseNumber()
		localPulse = global.GetLocal().GetPulseNumber()
	)
	return basePulse.IsSpecial() || basePulse.IsBeforeOrEq(localPulse)
}
