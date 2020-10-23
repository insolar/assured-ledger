// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapters

import (
	"context"

	transport2 "github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/transport"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l2/uniserver"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
)

type PacketSender struct {
	unifiedServer *uniserver.UnifiedServer
}

func NewPacketSender(unifiedServer *uniserver.UnifiedServer) *PacketSender {
	return &PacketSender{
		unifiedServer: unifiedServer,
	}
}

func (ps *PacketSender) SendPacketToTransport(ctx context.Context, to transport2.TargetProfile, sendOptions transport2.PacketSendOptions, payload interface{}) {
	addr := to.GetStatic().GetDefaultEndpoint().GetIPAddress().String()

	_, logger := inslogger.WithFields(ctx, map[string]interface{}{
		"receiver_addr": addr,
	})

	// nwapi.NewHostAndPort()
	peerAddr := nwapi.NewHostPort(addr, false)

	peer, err := ps.unifiedServer.PeerManager().Manager().ConnectPeer(peerAddr)
	if err != nil {
		logger.Error("Failed to connect to peer: ", err)
	}

	packet := &ConsensusPacket{Payload: payload.([]byte)}
	err = peer.SendPacket(uniproto.SessionlessNoQuota, packet)
	if err != nil {
		logger.Error("Failed to send datagram: ", err)
	}
}
