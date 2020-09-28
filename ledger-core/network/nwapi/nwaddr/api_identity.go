// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nwaddr

// Identity is a type of address identity
// It has a different package to avoid name collision
type Identity uint8

const (
	_ Identity = iota
	// IP is either IPv4 or IPv6
	IP
	// DNS is a symbolic name
	DNS
	// HostPK is a PK of a node
	HostPK
	// HostID is numeric id of a node
	HostID
	// LocalUID is a locally unique id assigned to a node. It changes each time when node is unregistered/disconnected.
	LocalUID
)

func (a Identity) IsIP() bool {
	return a == IP
}

func (a Identity) IsDNS() bool {
	return a == DNS
}

func (a Identity) IsHostPK() bool {
	return a == HostPK
}

func (a Identity) IsHostID() bool {
	return a == HostID
}

func (a Identity) IsLocalUID() bool {
	return a == LocalUID
}

// IsNetCompatible is true when address is compatible with "net" package
func (a Identity) IsNetCompatible() bool {
	return a == IP || a == DNS
}

// IsResolved is true when address is resolved and represents a single endpoint
func (a Identity) IsResolved() bool {
	return a == IP
}
