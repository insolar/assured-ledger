// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gen

import (
	fuzz "github.com/google/gofuzz"

	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

// Number generates random pulse number (excluding special cases).
func PulseNumber() pulse.Number {
	f := fuzz.New().NilChance(0).Funcs(func(pn *pulse.Number, c fuzz.Continue) {
		*pn = pulse.Number(c.Int31n(pulse.MaxTimePulse-pulse.MinTimePulse) + pulse.MinTimePulse)
	})

	var pn pulse.Number
	f.Fuzz(&pn)
	return pn
}
