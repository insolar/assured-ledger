// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsestor

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func ReadPulses(pulser StartPulse) func() {
	return func() {
		pulser.PulseNumber()
	}
}

func TestStartPulseRace(t *testing.T) {
	ctx := inslogger.TestContext(t)
	startPulse := NewStartPulse()
	for i := 0; i < 10; i++ {
		go ReadPulses(startPulse)()
	}
	startPulse.SetStartPulse(ctx, Pulse{PulseNumber: gen.PulseNumber()})
}
