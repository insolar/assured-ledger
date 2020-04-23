// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

type RegisterControllerFunc func(ProtocolType, Descriptor, Controller, Receiver)
type RegisterProtocolFunc func(RegisterControllerFunc)
type Controller interface {
	Start(PeerManager)
	NextPulse(pulse.Range)
	Stop()
}

type PeerManager interface {
	ConnectPeer(nwapi.Address) (Peer, error)
	ConnectedPeer(nwapi.Address) (Peer, error)
	MaxSmallPayloadSize() uint
	MaxSessionlessPayloadSize() uint
	LocalPeer() Peer
}
