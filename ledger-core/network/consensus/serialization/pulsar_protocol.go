package serialization

import (
	"context"
	"io"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/phases"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type PulsarPacketBody struct {
	// ByteSize>=108
	PulseNumber           pulse.Number  `insolar-transport:"ignore=send"`
	PulseDataExt          pulse.DataExt // ByteSize=44
	PulsarConsensusProofs []byte        // variable lengths >=0
}

func (b *PulsarPacketBody) String(ctx PacketContext) string {
	return "pulsar packet body"
}

func (b *PulsarPacketBody) SerializeTo(_ SerializeContext, writer io.Writer) error {
	if err := write(writer, b.PulseNumber); err != nil {
		return errors.W(err, "failed to serialize PulseNumber")
	}

	if err := write(writer, b.PulseDataExt); err != nil {
		return errors.W(err, "failed to serialize PulseDataExt")
	}

	return nil
}

func (b *PulsarPacketBody) DeserializeFrom(_ DeserializeContext, reader io.Reader) error {
	if err := read(reader, &b.PulseNumber); err != nil {
		return errors.W(err, "failed to deserialize PulseNumber")
	}

	if err := read(reader, &b.PulseDataExt); err != nil {
		return errors.W(err, "failed to deserialize PulseDataExt")
	}

	return nil
}

func (b *PulsarPacketBody) getPulseData() pulse.Data {
	return pulse.Data{
		PulseNumber: b.PulseNumber,
		DataExt:     b.PulseDataExt,
	}
}

func BuildPulsarPacket(ctc context.Context, pd pulse.Data) *Packet {
	packet := &Packet{}

	packet.Header.setProtocolType(ProtocolTypePulsar)
	packet.Header.setPacketType(phases.PacketPulsarPulse)
	packet.Header.setIsBodyEncrypted(true)
	packet.Header.SourceID = 1

	packet.setPulseNumber(pd.PulseNumber)
	packet.EncryptableBody = &PulsarPacketBody{
		PulseNumber:  pd.PulseNumber,
		PulseDataExt: pd.DataExt,
	}

	return packet
}
