// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodeinfo

import (
	"crypto"

	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/profiles"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

//go:generate stringer -type=State
type State uint8

const (
	_ State = iota
	// Joining node is in first pulse of discovery bootstrap or is joining to a bootstrapped network
	Joining
	// Ready node is connected to network
	Ready
)

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo.NetworkNode -o ../../testutils/network -s _mock.go -g

type NetworkNode interface {
	GetStatic() profiles.StaticProfile

	// ShortID get short ID of node
	GetNodeID() node.ShortNodeID

	// PublicKey is the public key of the node
	PublicKey() crypto.PublicKey

	GetDeclaredPower() member.Power

	IsJoiner() bool
	IsPowered() bool

	GetSignature() cryptkit.SignedDigestHolder
}

func NodeAddr(n NetworkNode) string {
	return n.GetStatic().GetDefaultEndpoint().GetIPAddress().String()
}

func NodeRef(n NetworkNode) reference.Global {
	return n.GetStatic().GetExtension().GetReference()
}

func NodeRole(n NetworkNode) member.PrimaryRole {
	return n.GetStatic().GetPrimaryRole()
}
