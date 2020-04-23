// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import "math"

type Dispatcher interface {
	Controller

	GetMode() ConnectionMode
	GetReceiver(pt ProtocolType) Receiver
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
