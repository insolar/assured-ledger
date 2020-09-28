// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

// RegisterControllerFunc is provided to register a protocol.
// For each protocol there should be provided ProtocolType, a Descriptor with protocol's packets and feature, a Controller and Receiver.
type RegisterControllerFunc func(ProtocolType, Descriptor, Controller, Receiver)
type RegisterProtocolFunc func(RegisterControllerFunc)

// Controller is an interface for life-cycle of a protocol or a set of protocols.
type Controller interface {
	// Start is called when protocol(s) should be activated. Individual protocols can be started or stopped multiple times.
	Start(PeerManager)
	// NextPulse is called when a next pulse is committed. Must be fast, can trigger housekeeping etc.
	// Also NextPulse has to be called immediately after Start with the latest pulse.Range when applicable.
	NextPulse(pulse.Range)
	// Start is called when protocol(s) should be deactivated. Individual protocols can be started or stopped multiple times.
	Stop()
}

// PeerManager provides access to a set of peers.
type PeerManager interface {
	// ConnectPeer either returns a known peer for the given address. Or attempts to connect the given address with EnsureConnect().
	// A new Peer will not be added if an error occur. For local peer will return (nil, nil).
	ConnectPeer(nwapi.Address) (Peer, error)
	// ConnectedPeer returns an already known peer for the given address.
	// Returns error when peer is unknown. For local peer will return (nil, nil).
	ConnectedPeer(nwapi.Address) (Peer, error)

	// MaxSessionlessPayloadSize returns a max size of a packet allowed for sessionless delivery.
	MaxSessionlessPayloadSize() uint
	// MaxSessionlessPayloadSize returns a max size of a packet allowed for small session-based delivery.
	MaxSmallPayloadSize() uint
	// LocalPeer returns a peer representing the local host.
	LocalPeer() Peer
}
