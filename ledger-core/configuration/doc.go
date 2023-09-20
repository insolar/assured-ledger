/*
Package configuration holds configuration for all components in Insolar host binary
It allows also helps to manage config resources using Holder

Usage:

	package main

	import (
		"github.com/insolar/assured-ledger/ledger-core/configuration"
		"fmt"
	)

	func main() {
		holder := configuration.NewHolder("insolar.yml")
		holder.MustLoad()
		fmt.Printf("Default configuration:\n %+v\n", holder.Configuration)
	}

*/
package configuration
