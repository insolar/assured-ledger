// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/apinetwork"
)

type ProtocolController struct {
}

var _ apinetwork.ProtocolReceiver = &receiver{}

type receiver struct {
}

func (p *receiver) ReceiveSmallPacket(from apinetwork.Address, packet apinetwork.Packet, b []byte, sigLen uint32) {

}

func (p *receiver) ReceiveLargePacket(from apinetwork.Address, packet apinetwork.Packet, preRead []byte, r io.LimitedReader, verifier apinetwork.PacketDataVerifier) error {
	panic("implement me")
}
