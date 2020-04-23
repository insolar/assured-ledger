// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/uniproto"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

// Flags for StatePacket
const (
	BodyRqFlag uniproto.FlagIndex = iota
	BodyAckListFlag
	RejectListFlag
)

var _ uniproto.ProtocolPacket = &ParcelPacket{}

type StatePacket struct {
	// From receiver to sender
	BodyRq []ShortShipmentID `insolar-transport:"optional=PacketFlags[0]"` //
	// From receiver to sender
	BodyAckList []ShortShipmentID `insolar-transport:"optional=PacketFlags[1]"` // TODO serialization / deser
	// From receiver to sender
	RejectList []ShortShipmentID `insolar-transport:"optional=PacketFlags[2]"` //
	// From receiver to sender
	AckList []ShortShipmentID `insolar-transport:"list=nocount"` // length is determined by packet size
}

func (p *StatePacket) remainingSpace(maxSize int) int {
	total := 0
	if n := len(p.BodyRq); n > 0 {
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
	return len(p.AckList) == 0 && len(p.BodyRq) == 0 && len(p.BodyAckList) == 0 && len(p.RejectList) == 0
}

func (p *StatePacket) PreparePacket() (packet uniproto.PacketTemplate, dataSize uint, fn uniproto.PayloadSerializerFunc) {
	initPacket(DeliveryState, &packet.Packet)

	if n := len(p.BodyRq); n > 0 {
		packet.Header.SetFlag(BodyRqFlag, true)
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

func listToBytes(list []ShortShipmentID, b []byte) ([]byte, error) {
	for _, id := range list {
		if id == 0 {
			return nil, throw.IllegalValue()
		}
		if len(b) < ShortShipmentIDByteSize {
			return nil, io.ErrShortBuffer
		}
		id.PutTo(b)
		b = b[ShortShipmentIDByteSize:]
	}
	return b, nil
}

func countedListToBytes(list []ShortShipmentID, b []byte) ([]byte, error) {
	sz := uint(protokit.EncodeVarintToBytes(b, uint64(len(list))))
	return listToBytes(list, b[sz:])
}

func (p *StatePacket) SerializePayload(_ nwapi.SerializationContext, _ *uniproto.Packet, writer *iokit.LimitedWriter) (err error) {
	out := make([]byte, writer.RemainingBytes())
	b := out

	if list := p.BodyRq; len(list) > 0 {
		if b, err = countedListToBytes(list, b); err != nil {
			return err
		}
	}

	if list := p.BodyAckList; len(list) > 0 {
		if b, err = countedListToBytes(list, b); err != nil {
			return err
		}
	}

	if list := p.RejectList; len(list) > 0 {
		if b, err = countedListToBytes(list, b); err != nil {
			return err
		}
	}

	if b, err = listToBytes(p.AckList, b); err != nil {
		return err
	}

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

	if packet.Header.HasFlag(BodyRqFlag) {
		if b, p.BodyRq, err = readCountedList(b); err != nil {
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
