package main

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/example"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/convlog"
)

func main() {
	example.RunExample(convlog.MachineLogger{})
}
