package main

import (
	"github.com/insolar/assured-ledger/ledger-core/load"
	"github.com/skudasov/loadgen"
)

func main() {
	loadgen.Run(load.AttackerFromName, load.CheckFromName)
}
