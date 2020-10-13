// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package packet

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func DeserializePacketRaw(conn io.Reader) (*ReceivedPacket, uint64, error) {
	reader := &capturingReader{Reader: conn}

	lengthBytes := make([]byte, 8)
	if _, err := io.ReadFull(reader, lengthBytes); err != nil {
		return nil, 0, err
	}
	lengthReader := bytes.NewReader(lengthBytes)
	length, err := binary.ReadUvarint(lengthReader)
	if err != nil {
		return nil, 0, io.ErrUnexpectedEOF
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return nil, 0, errors.W(err, "failed to read packet")
	}

	msg := &rms.Packet{}
	err = msg.Unmarshal(buf)
	if err != nil {
		return nil, 0, errors.W(err, "failed to decode packet")
	}

	receivedPacket := NewReceivedPacket(msg, reader.Captured())
	return receivedPacket, length, nil
}

// DeserializePacket reads packet from io.Reader.
func DeserializePacket(logger log.Logger, conn io.Reader) (*ReceivedPacket, uint64, error) {
	receivedPacket, length, err := DeserializePacketRaw(conn)
	if err != nil {
		return nil, 0, err
	}
	logger.Debugf("[ DeserializePacket ] decoded packet to %s", receivedPacket.DebugString())
	return receivedPacket, length, nil
}

func NewPacket(sender, receiver nwapi.Address, packetType types.PacketType, id uint64) *rms.Packet {
	return &rms.Packet{
		// Polymorph field should be non-default so we have first byte 0x80 in serialized representation
		Polymorph: 1,
		Sender:    sender,
		Receiver:  receiver,
		Type:      uint32(packetType),
		RequestID: id,
	}
}

type capturingReader struct {
	io.Reader
	buffer bytes.Buffer
}

func (r *capturingReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.buffer.Write(p)
	return n, err
}

func (r *capturingReader) Captured() []byte {
	return r.buffer.Bytes()
}

// SerializePacket converts packet to byte slice.
func SerializePacket(p *rms.Packet) ([]byte, error) {
	data, err := p.Marshal()
	if err != nil {
		return nil, errors.W(err, "Failed to serialize packet")
	}

	var lengthBytes [8]byte
	binary.PutUvarint(lengthBytes[:], uint64(p.ProtoSize()))

	var result []byte
	result = append(result, lengthBytes[:]...)
	result = append(result, data...)

	return result, nil
}
