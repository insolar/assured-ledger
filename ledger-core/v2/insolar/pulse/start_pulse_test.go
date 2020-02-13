// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulse

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
)

func ReadPulses(t testing.TB, pulser StartPulse) func() {
	return func() {
		pulser.PulseNumber()
	}
}

func TestStartPulseRace(t *testing.T) {
	ctx := inslogger.TestContext(t)
	startPulse := NewStartPulse()
	syncTest := &testutils.SyncT{TB: t}
	for i := 0; i < 10; i++ {
		go ReadPulses(syncTest, startPulse)()
	}
	startPulse.SetStartPulse(ctx, insolar.Pulse{PulseNumber: gen.PulseNumber()})
}
