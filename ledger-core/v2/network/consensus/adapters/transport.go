// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapters

import (
	"context"

	transport2 "github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/transport"

	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/transport"
)

type PacketSender struct {
	datagramTransport transport.DatagramTransport
}

func NewPacketSender(datagramTransport transport.DatagramTransport) *PacketSender {
	return &PacketSender{
		datagramTransport: datagramTransport,
	}
}

func (ps *PacketSender) SendPacketToTransport(ctx context.Context, to transport2.TargetProfile, sendOptions transport2.PacketSendOptions, payload interface{}) {
	addr := to.GetStatic().GetDefaultEndpoint().GetIPAddress().String()

	ctx, logger := inslogger.WithFields(ctx, map[string]interface{}{
		"receiver_addr": addr,
	})

	err := ps.datagramTransport.SendDatagram(ctx, addr, payload.([]byte))
	if err != nil {
		logger.Error("Failed to send datagram: ", err)
	}
}
