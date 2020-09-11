// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package configuration

// Virtual holds configuration for ledger.
type Virtual struct {
	// MaxRunners limits number of contract executions running in parallel.
	// If set to zero or a negative value, limit will be set automatically to
	// `( runtime.NumCPU() - 2 ) but not less then 4`.
	MaxRunners int
}

// NewLedger creates new default Ledger configuration.
func NewVirtual() Virtual {
	return Virtual{
		MaxRunners: 0, // auto
	}
}
