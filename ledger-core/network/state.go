// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package network

// State type for bootstrapping process
type State int

//go:generate stringer -type=State
const (
	// NoNetworkState state means that nodes doesn`t match majority_rule
	NoNetworkState State = iota
	JoinerBootstrap
	DiscoveryBootstrap
	WaitConsensus
	WaitMajority
	WaitMinRoles
	WaitPulsar
	CompleteNetworkState
)

