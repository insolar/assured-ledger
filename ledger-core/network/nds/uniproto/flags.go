// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

// Flags is flags of a packet type of a protocol.
type Flags uint8

const (
	// DatagramOnly allows receiving of this packet type ONLY over a sessionless connection.
	DatagramOnly Flags = 1 << iota
	// DatagramOnly allows receiving of this packet type over a sessionless connection.
	DatagramAllowed
	// DisableRelay disallows packet relays. Only packets delivered directly will be accepted.
	DisableRelay
	// OmitSignatureOverTLS omits packet signature(s) for direct packets when TLS connection is in use.
	// Use of this flag will deny TLS connections with unrecognized certificates.
	OmitSignatureOverTLS
	// SourcePK indicates presence of source's PK in the packet header. And PK can be used to verify packet.
	SourcePK
	// OptionalTarget allows non-specified target for this packet type. E.g. to send to an unknown or to multiple peers.
	OptionalTarget
	// NoSourceID requires source id to be 0 (unspecified). Applied for requests from external / unknown nodes.
	NoSourceID
)
