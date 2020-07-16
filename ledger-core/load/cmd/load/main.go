package main

import (
	"github.com/insolar/assured-ledger/ledger-core/load"
	"github.com/insolar/loadgenerator"
)

func main() {
	loadgen.Run(load.AttackerFromName, load.CheckFromName, nil, nil)
}
