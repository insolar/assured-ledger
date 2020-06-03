// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

/*
Package configuration holds configuration for all components in Insolar host binary
It allows also helps to manage config resources using Holder

Usage:

	package main

	import (
		"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
		"fmt"
	)

	func main() {
		holder := configuration.NewHolder("insolar.yml")
		holder.MustLoad()
		fmt.Printf("Default configuration:\n %+v\n", holder.Configuration)
	}

*/
package configuration
