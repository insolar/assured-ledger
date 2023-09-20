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
