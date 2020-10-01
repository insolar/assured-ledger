// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulse

// Number is a type for pulse numbers.
//
// Special values:
//     0 				Unknown
//     1 .. 256			RESERVED. For package internal usage
//
//   257 .. 65279		Available for platform-wide usage
//
// 65280 .. 65535		RESERVED. For global maps and aliases
// 65536				Local relative pulse number
// 65537 .. 1<<30 - 1	Regular time based pulse numbers
//
// NB! Range 0..256 IS RESERVED for internal operations
// There MUST BE NO references with PN < 256 ever visible to contracts / users.
const (
	_ Number = 256 + iota

	// Jet is a special pulse number value that signifies jet ID.
	// TODO either JetPrefix or ShortJetId - both are viable for addressing
	// Local part (can be omitted) - then it is a reference to a specific record within the jet
	// or to a jet-local built-in contract (via Base part)
	Jet

	// BuiltinContract declares special pulse number that creates namespace for builtin contracts
	// Base part is type/contract identity, Local part (can be omitted) identifies a version
	BuiltinContract

	// Base part - see FullJetId, and it has an indication to represent a JetDrop reference.
	// Local part (can be omitted) - then it is a reference to a specific record within the jet
	JetGeneration // and JetDrop and JetContract

	// Node, it is identified by 224 bits of node's PK hash
	// Local part (can be omitted) - then it is a reference to a specific state of the node or
	// node-local built-in contracts
	Node

	// Reference to a part of lifeline's record that is reused within the same lifeline without copying.
	// Base part defines jet + position of a referenced content within the record
	// Local part is a record id within the relevant lifeline/jet
	RecordPayload

	// Identity of an external call - initially it is not bound to lifelines, hence the separate addressing.
	// Base part - same as of Node ref of the node accepted the call
	// Local part - pulse of seed, hash of request
	ExternalCall

	// Identity of an endpoint
	// Indicates when the endpoint is free-to-call, can be served by any node, or needs a special handling.
	EndpointAddress
	// TollFreeEndpointAddress ??

	// Identity of data relevant to pulse, e.g. network state hash, network population, jet tree etc
	DataOfPulse
)
