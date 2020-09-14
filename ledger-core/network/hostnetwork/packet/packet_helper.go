// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package packet

import (
	"bytes"
	"encoding/binary"
	"io"
	"strconv"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/host"
	"github.com/insolar/assured-ledger/ledger-core/network/hostnetwork/packet/types"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

func (p *Packet) SetRequest(request interface{}) {
	var r isRequest_Request
	switch t := request.(type) {
	case *RPCRequest:
		r = &Request_RPC{t}
	case *PulseRequest:
		r = &Request_Pulse{t}
	case *BootstrapRequest:
		r = &Request_Bootstrap{t}
	case *AuthorizeRequest:
		r = &Request_Authorize{t}
	case *SignCertRequest:
		r = &Request_SignCert{t}
	case *UpdateScheduleRequest:
		r = &Request_UpdateSchedule{t}
	case *ReconnectRequest:
		r = &Request_Reconnect{t}
	default:
		panic("Request payload is not a valid protobuf struct!")
	}
	p.Payload = &Packet_Request{Request: &Request{Request: r}}
}

func (p *Packet) SetResponse(response interface{}) {
	var r isResponse_Response
	switch t := response.(type) {
	case *RPCResponse:
		r = &Response_RPC{t}
	case *BasicResponse:
		r = &Response_Basic{t}
	case *BootstrapResponse:
		r = &Response_Bootstrap{t}
	case *AuthorizeResponse:
		r = &Response_Authorize{t}
	case *SignCertResponse:
		r = &Response_SignCert{t}
	case *ErrorResponse:
		r = &Response_Error{t}
	case *UpdateScheduleResponse:
		r = &Response_UpdateSchedule{t}
	case *ReconnectResponse:
		r = &Response_Reconnect{t}
	default:
		panic("Response payload is not a valid protobuf struct!")
	}
	p.Payload = &Packet_Response{Response: &Response{Response: r}}
}

func (p *Packet) GetType() types.PacketType {
	// TODO: make p.Type of type PacketType instead of uint32
	return types.PacketType(p.Type)
}

func (p *Packet) GetSender() reference.Global {
	return p.Sender.NodeID
}

func (p *Packet) GetSenderHost() *host.Host {
	return p.Sender
}

func (p *Packet) GetRequestID() types.RequestID {
	return types.RequestID(p.RequestID)
}

func (p *Packet) IsResponse() bool {
	return p.GetResponse() != nil
}

// SerializePacket converts packet to byte slice.
func SerializePacket(p *Packet) ([]byte, error) {
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

func DeserializePacketRaw(conn io.Reader) (*ReceivedPacket, uint64, error) {
	reader := NewCapturingReader(conn)

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

	msg := &Packet{}
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

func (p *Packet) DebugString() string {
	if p == nil {
		return "nil"
	}
	return `&Packet{` +
		`Sender:` + p.Sender.String() + `,` +
		`Receiver:` + p.Receiver.String() + `,` +
		`RequestID:` + strconv.FormatUint(p.RequestID, 10) + `,` +
		`TraceID:` + p.TraceID + `,` +
		`Type:` + p.GetType().String() + `,` +
		`IsResponse:` + strconv.FormatBool(p.IsResponse()) + `,` +
		`}`
}

func NewPacket(sender, receiver *host.Host, packetType types.PacketType, id uint64) *Packet {
	return &Packet{
		// Polymorph field should be non-default so we have first byte 0x80 in serialized representation
		Polymorph: 1,
		Sender:    sender,
		Receiver:  receiver,
		Type:      uint32(packetType),
		RequestID: id,
	}
}

type CapturingReader struct {
	io.Reader
	buffer bytes.Buffer
}

func NewCapturingReader(reader io.Reader) *CapturingReader {
	return &CapturingReader{Reader: reader}
}

func (r *CapturingReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.buffer.Write(p)
	return n, err
}

func (r *CapturingReader) Captured() []byte {
	return r.buffer.Bytes()
}