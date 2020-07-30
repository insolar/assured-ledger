// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nodeset

import (
	"net"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/adapters"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/censusimpl"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/resolver"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// NewNodeNetwork create active node component
func NewNodeNetwork(configuration configuration.Transport, certificate nodeinfo.Certificate) (network.NodeNetwork, error) { // nolint:staticcheck
	origin, err := createOrigin(configuration, certificate)
	if err != nil {
		return nil, throw.W(err, "Failed to create origin node")
	}
	nodeKeeper := NewNodeKeeper(origin)
	return nodeKeeper, nil
}

func createOrigin(configuration configuration.Transport, cert nodeinfo.Certificate) (nodeinfo.NetworkNode, error) {
	publicAddress, err := resolveAddress(configuration)
	if err != nil {
		return nil, throw.W(err, "Failed to resolve public address")
	}

	role := cert.GetRole()
	if role == member.PrimaryRoleUnknown {
		panic(throw.IllegalValue())
	}

	ref := cert.GetNodeRef()

	specialRole := member.SpecialRoleNone
	isDiscovery := network.IsDiscovery(ref, cert)
	if isDiscovery {
		specialRole = member.SpecialRoleDiscovery
	}

	pk := cert.GetPublicKey()
	nodeID := node.ShortNodeID(node.GenerateUintShortID(ref))
	staticExt := adapters.NewStaticProfileExtensionExt(nodeID, ref, cryptkit.Signature{})

	staticProfile := adapters.NewStaticProfileExt2(
		node.ShortNodeID(node.GenerateUintShortID(ref)), role, specialRole,
		staticExt,
		adapters.NewOutboundNoPort(publicAddress),
		adapters.ECDSAPublicKeyAsPublicKeyStore(pk),
		nil, cryptkit.SignedDigest{},
	)

	var verifier cryptkit.SignatureVerifier // = nil
	var anp censusimpl.NodeProfileSlot
	if isDiscovery {
		anp = censusimpl.NewNodeProfile(0, staticProfile, verifier, staticProfile.GetStartPower())
	} else {
		anp = censusimpl.NewJoinerProfile(staticProfile, verifier)
	}
	return adapters.NewNetworkNode(&anp), nil
}

func resolveAddress(configuration configuration.Transport) (string, error) {
	addr, err := net.ResolveTCPAddr("tcp", configuration.Address)
	if err != nil {
		return "", err
	}
	address, err := resolver.Resolve(configuration.FixedPublicAddress, addr.String())
	if err != nil {
		return "", err
	}
	return address, nil
}

