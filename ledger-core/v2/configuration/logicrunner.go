// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package configuration

// LogicRunner configuration
type LogicRunner struct {
	// RPCListen - address logic runner binds RPC API to
	RPCListen string
	// RPCProtoco - protocol (network) of above address,
	// e.g. "tcp", "unix"... see `net.Dial`
	RPCProtocol string
	// GoPlugin - configuration of executor based on Go plugins
	GoPlugin *GoPlugin
	// PulseLRUSize - configuration of size of a pulse's cache
	PulseLRUSize int
}

// GoPlugin configuration
type GoPlugin struct {
	// RunnerListen - address Go plugins executor listens to
	RunnerListen string
	// RunnerProtocol - protocol (network) of above address,
	// e.g. "tcp", "unix"... see `net.Dial`
	RunnerProtocol string
}

// NewLogicRunner - returns default config of the logic runner
func NewLogicRunner() LogicRunner {
	return LogicRunner{
		RPCListen:   "127.0.0.1:7778",
		RPCProtocol: "tcp",
		GoPlugin: &GoPlugin{
			RunnerListen:   "127.0.0.1:7777",
			RunnerProtocol: "tcp",
		},
		PulseLRUSize: 100,
	}
}
