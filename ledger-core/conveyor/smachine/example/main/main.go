// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/example"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/convlog"
)

func main() {
	example.RunExample(convlog.MachineLogger{})
}
