package adapters

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
)

var _ uniproto.ProtocolPacket = &ConsensusPacket{}

type ConsensusPacket struct {
	Payload []byte
}

func (p *ConsensusPacket) PreparePacket() (uniproto.PacketTemplate, uint, uniproto.PayloadSerializerFunc) {
	pt := uniproto.PacketTemplate{}
	pt.Header.SetRelayRestricted(true)
	pt.Header.SetProtocolType(uniproto.ProtocolTypePulsar)
	pt.PulseNumber = pulse.MinTimePulse
	return pt, uint(len(p.Payload)), p.SerializePayload
}

func (p *ConsensusPacket) SerializePayload(_ nwapi.SerializationContext, _ *uniproto.Packet, writer *iokit.LimitedWriter) error {
	_, err := writer.Write(p.Payload)
	return err
}

func (p *ConsensusPacket) DeserializePayload(_ nwapi.DeserializationContext, _ *uniproto.Packet, reader *iokit.LimitedReader) error {
	b := make([]byte, reader.RemainingBytes())
	_, err := io.ReadFull(reader, b)
	p.Payload = b
	return err
}
