package example

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/convlog"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insconveyor"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
)

func TestExample(t *testing.T) {
	instestlogger.SetTestOutput(t)

	var machineLogger smachine.SlotMachineLogger
	if convlog.UseTextConvLog() {
		machineLogger = convlog.MachineLogger{}
	} else {
		machineLogger = insconveyor.ConveyorLoggerFactory{}
	}

	RunExample(machineLogger)
}
