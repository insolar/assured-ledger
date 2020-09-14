// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package packet

import (
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

type ReceivedPacket struct {
	*rms.Packet
	data []byte
}

func NewReceivedPacket(p *rms.Packet, data []byte) *ReceivedPacket {
	return &ReceivedPacket{
		Packet: p,
		data:   data,
	}
}

func (p *ReceivedPacket) Bytes() []byte {
	return p.data
}
