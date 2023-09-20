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

