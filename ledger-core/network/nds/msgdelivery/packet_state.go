package msgdelivery

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

// Flags for StatePacket
const (
	BodyRqListFlag uniproto.FlagIndex = iota
	BodyAckListFlag
	RejectListFlag
)

var _ uniproto.ProtocolPacket = &ParcelPacket{}

type StatePacket struct {
	// From receiver to sender
	BodyRqList []ShortShipmentID `insolar-transport:"optional=PacketFlags[0]"` //
	// From receiver to sender
	BodyAckList []ShortShipmentID `insolar-transport:"optional=PacketFlags[1]"` // TODO serialization / deser
	// From receiver to sender
	RejectList []ShortShipmentID `insolar-transport:"optional=PacketFlags[2]"` //
	// From receiver to sender
	AckList []ShortShipmentID `insolar-transport:"list=nocount"` // length is determined by packet size
}

func (p *StatePacket) remainingSpace(maxSize int) int {
	total := 0
	if n := len(p.BodyRqList); n > 0 {
		total += n + 1 // +1 for list length
	}
	if n := len(p.BodyAckList); n > 0 {
		total += n + 1 // +1 for list length
	}
	if n := len(p.RejectList); n > 0 {
		total += n + 1 // +1 for list length
	}
	if n := len(p.AckList); n > 0 {
		total += n
	}
	return maxSize/ShortShipmentIDByteSize - total
}

func (p *StatePacket) isEmpty() bool {
	return len(p.AckList) == 0 && len(p.BodyRqList) == 0 && len(p.BodyAckList) == 0 && len(p.RejectList) == 0
}

func (p *StatePacket) PreparePacket() (packet uniproto.PacketTemplate, dataSize uint, fn uniproto.PayloadSerializerFunc) {
	initPacket(DeliveryState, &packet.Packet)

	if n := len(p.BodyRqList); n > 0 {
		packet.Header.SetFlag(BodyRqListFlag, true)
		dataSize += uint(protokit.SizeVarint64(uint64(n)))
		dataSize += ShortShipmentIDByteSize * uint(n)
	}
	if n := len(p.BodyAckList); n > 0 {
		packet.Header.SetFlag(BodyAckListFlag, true)
		dataSize += uint(protokit.SizeVarint64(uint64(n)))
		dataSize += ShortShipmentIDByteSize * uint(n)
	}
	if n := len(p.RejectList); n > 0 {
		packet.Header.SetFlag(RejectListFlag, true)
		dataSize += uint(protokit.SizeVarint64(uint64(n)))
		dataSize += ShortShipmentIDByteSize * uint(n)
	}
	dataSize += uint(len(p.AckList)) * ShortShipmentIDByteSize

	return packet, dataSize, p.SerializePayload
}

func listToBytes(list []ShortShipmentID, b []byte) []byte {
	for _, id := range list {
		if id == 0 {
			panic(throw.IllegalValue())
		}
		id.PutTo(b)
		b = b[ShortShipmentIDByteSize:]
	}
	return b
}

func countedListToBytes(list []ShortShipmentID, b []byte) []byte {
	sz := uint(protokit.EncodeVarintToBytes(b, uint64(len(list))))
	return listToBytes(list, b[sz:])
}

func (p *StatePacket) SerializePayload(_ nwapi.SerializationContext, _ *uniproto.Packet, writer *iokit.LimitedWriter) (err error) {
	out := make([]byte, writer.RemainingBytes())
	b := out

	if list := p.BodyRqList; len(list) > 0 {
		b = countedListToBytes(list, b)
	}

	if list := p.BodyAckList; len(list) > 0 {
		b = countedListToBytes(list, b)
	}

	if list := p.RejectList; len(list) > 0 {
		b = countedListToBytes(list, b)
	}

	b = listToBytes(p.AckList, b)

	_, err = writer.Write(out[:len(out)-len(b)])
	return err
}

func readCountedList(b []byte) ([]byte, []ShortShipmentID, error) {
	count, n := protokit.DecodeVarintFromBytes(b)
	if count == 0 || n == 0 {
		return b, nil, throw.IllegalValue()
	}
	b = b[n:]
	if count*ShortShipmentIDByteSize > uint64(len(b)) {
		return b, nil, throw.IllegalValue()
	}
	ids := make([]ShortShipmentID, count)
	var err error
	if b, err = _readList(b, ids); err != nil {
		return b, nil, err
	}
	return b, ids, nil
}

func readList(b []byte) ([]ShortShipmentID, error) {
	count := len(b) / ShortShipmentIDByteSize
	if len(b)%ShortShipmentIDByteSize != 0 {
		return nil, throw.IllegalValue()
	}
	ids := make([]ShortShipmentID, count)
	if _, err := _readList(b, ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func _readList(b []byte, ids []ShortShipmentID) ([]byte, error) {
	for i := range ids {
		id := ShortShipmentIDReadFromBytes(b)
		if id == 0 {
			return nil, throw.IllegalValue()
		}
		ids[i] = id
		b = b[ShortShipmentIDByteSize:]
	}
	return b, nil
}

func (p *StatePacket) DeserializePayload(_ nwapi.DeserializationContext, packet *uniproto.Packet, reader *iokit.LimitedReader) (err error) {
	if reader.RemainingBytes() < ShortShipmentIDByteSize {
		return throw.IllegalValue()
	}

	b := make([]byte, reader.RemainingBytes())
	if _, err = reader.Read(b); err != nil {
		return err
	}

	if packet.Header.HasFlag(BodyRqListFlag) {
		if b, p.BodyRqList, err = readCountedList(b); err != nil {
			return err
		}
	}

	if packet.Header.HasFlag(BodyAckListFlag) {
		if b, p.BodyAckList, err = readCountedList(b); err != nil {
			return err
		}
	}

	if packet.Header.HasFlag(RejectListFlag) {
		if b, p.RejectList, err = readCountedList(b); err != nil {
			return err
		}
	}

	p.AckList, err = readList(b)
	return err
}
