// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package example

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/convlog"
	"github.com/insolar/assured-ledger/ledger-core/virtual/statemachine"
)

func TestExample(t *testing.T) {
	instestlogger.SetTestOutput(t)

	var machineLogger smachine.SlotMachineLogger
	if convlog.UseTextConvLog {
		machineLogger = convlog.MachineLogger{}
	} else {
		machineLogger = statemachine.ConveyorLoggerFactory{}
	}

	RunExample(machineLogger)
}
