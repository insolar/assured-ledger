// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import "math"

// Dispatcher provides life-cycle management for a set of protocols.
// Dispatcher can only be started (and stopped) once.
type Dispatcher interface {
	Controller

	// GetMode returns information about allowed protocols and flags.
	GetMode() ConnectionMode
	// GetReceiver returns a Receiver for the given protocol. Will return nil when the protocol is not know / not supported.
	GetReceiver(pt ProtocolType) Receiver
	// Seal should return a list of protocol's descriptors known to this Dispatcher. No changes are allowed after this call.
	Seal() Descriptors
}

func NewConnectionMode(m ConnectionMode, pts ...ProtocolType) ConnectionMode {
	return m.SetAllowedSet(NewProtocolSet(pts...))
}

type ConnectionMode uint32

const (
	AllowUnknownPeer ConnectionMode = 1 << iota
)

const AllowAll = ^ConnectionMode(0)

func (v ConnectionMode) IsProtocolAllowed(pt ProtocolType) bool {
	return v.AllowedSet().Has(pt)
}

func (v ConnectionMode) IsUnknownPeerAllowed() bool {
	return v&AllowUnknownPeer != 0
}

func (v ConnectionMode) AllowedSet() ProtocolSet {
	return ProtocolSet(v >> 16)
}

func (v ConnectionMode) SetAllowedSet(s ProtocolSet) ConnectionMode {
	return v&math.MaxUint16 | ConnectionMode(s)<<16
}
