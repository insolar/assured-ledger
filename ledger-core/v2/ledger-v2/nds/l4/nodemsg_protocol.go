// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package l4

import (
	"encoding/binary"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/network/apinetwork"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

const ProtocolNodeMessage = apinetwork.ProtocolTypeNodeMessage

type NodeMessageType uint8

const (
	NodeMessageState NodeMessageType = iota
	NodeMessageParcelHead
	NodeMessageParcelBody // or Head + Body
)

func NewPacket(tp NodeMessageType) Packet {
	pt := Packet{}
	pt.Header.SetProtocolType(ProtocolNodeMessage)
	pt.Header.SetPacketType(uint8(tp))
	return pt
}

type Packet struct {
	Header      apinetwork.Header `insolar-transport:"Protocol=0x03;Packet=0-1"`             // ByteSize=16
	PulseNumber pulse.Number      `insolar-transport:"[30-31]=0"`                            // [30-31] MUST ==0, ByteSize=4
	ExtraLength uint32            `insolar-transport:"Packet=1-;optional=IsExcessiveLength"` // ByteSize=4

	EncryptionData []byte `insolar-transport:"Packet=1;optional=IsBodyEncrypted"`
	// EncryptableBody interface{}

	PacketSignature longbits.Bits512 `insolar-transport:"generate=signature"` // ByteSize=64
}

func (p *Packet) SerializeTo(ctx apinetwork.SerializationContext, writer io.Writer, dataSize uint, fn func(*iokit.LimitedWriter) error) error {
	//p.Header.SetProtocolType(ProtocolNodeMessage)

	if pn, err := ctx.PrepareHeader(&p.Header); err != nil {
		return err
	} else if p.PulseNumber == 0 {
		p.PulseNumber = pn
	}

	signer := ctx.GetPayloadSigner()
	hasher := signer.NewHasher()

	packetSize := dataSize
	packetSize += uint(pulse.NumberSize)
	packetSize += uint(signer.GetDigestSize())

	if p.Header.IsBodyEncrypted() {
		// TODO encryption data size calc
		packetSize += 0
	}

	p.Header.SetPayloadLength(uint64(packetSize))
	if p.Header.IsExcessiveLength() && NodeMessageType(p.Header.GetPacketType()) == NodeMessageState {
		// ExtraLength is not allowed for packet type 0
		return throw.IllegalValue()
	}
	packetSize += p.Header.ByteSize()

	teeWriter := iokit.NewLimitedTeeWriterWithWipe(writer, hasher, p.Header.GetHashingZeroPrefix(), int64(packetSize))

	if err := p.Header.SerializeTo(teeWriter); err != nil {
		return err
	}

	byteOrder := binary.LittleEndian
	buf := make([]byte, pulse.NumberSize)
	byteOrder.PutUint32(buf, uint32(p.PulseNumber))
	if _, err := teeWriter.Write(buf); err != nil {
		return err
	}
	if p.ExtraLength > 0 {
		byteOrder.PutUint32(buf, p.ExtraLength)
		if _, err := teeWriter.Write(buf[:4]); err != nil {
			return err
		}
	}

	if p.Header.IsBodyEncrypted() {
		encWriter := ctx.GetPayloadEncrypter().NewEncryptingWriter(teeWriter)
		teeEncWriter := iokit.LimitWriter(encWriter, teeWriter.RemainingBytes())

		if err := fn(teeEncWriter); err != nil {
			return err
		}
	} else {
		if err := fn(teeWriter); err != nil {
			return err
		}
	}
	if teeWriter.RemainingBytes() != 0 {
		return throw.IllegalState()
	}

	digest := hasher.SumToDigest()
	signature := signer.SignDigest(digest)

	if err := longbits.CopyAllBytes(signature, p.PacketSignature[:]); err != nil {
		return err
	}
	_, err := signature.WriteTo(writer)
	return err
}

func (p *Packet) DeserializeFrom(ctx apinetwork.DeserializationContext, reader io.Reader, fn func(*iokit.LimitedReader) error) error {
	signer := ctx.GetPayloadVerifier()
	hasher := signer.NewHasher()

	teeReader := iokit.NewTeeReaderWithWipe(reader, hasher, p.Header.GetHashingZeroPrefix())
	if err := p.Header.DeserializeFrom(teeReader); err != nil {
		return err
	}

	if p.Header.GetProtocolType() != ProtocolNodeMessage {
		return throw.IllegalValue()
	}

	byteOrder := binary.LittleEndian
	buf := make([]byte, 4)
	if _, err := reader.Read(buf); err != nil {
		return err
	}
	if n := byteOrder.Uint32(buf); pulse.IsValidAsPulseNumber(int(n)) {
		p.PulseNumber = pulse.Number(n)
	} else {
		return throw.IllegalValue()
	}

	if err := ctx.VerifyHeader(&p.Header, p.PulseNumber); err != nil {
		return err
	}

	readLimit := int64(0)
	switch limit, err := p.Header.GetPayloadLength(); {
	case err != nil:
		return err
	case p.Header.IsExcessiveLength() && NodeMessageType(p.Header.GetPacketType()) == NodeMessageState:
		return throw.IllegalValue()
	default:
		readLimit = int64(limit)
	}

	switch n := int64(signer.GetDigestSize()); {
	case readLimit < n:
		return throw.IllegalValue()
	case n != int64(len(p.PacketSignature)):
		return throw.IllegalValue()
	default:
		readLimit -= n
	}

	if readLimit > 0 && p.Header.IsBodyEncrypted() {
		encReader := ctx.GetPayloadDecrypter().NewDecryptingReader(teeReader)
		limitReader := iokit.LimitReader(encReader, readLimit)
		if err := fn(limitReader); err != nil {
			return err
		}
		readLimit = limitReader.RemainingBytes()
	} else {
		limitReader := iokit.LimitReader(teeReader, readLimit)
		if err := fn(limitReader); err != nil {
			return err
		}
		readLimit = limitReader.RemainingBytes()
	}
	if readLimit != 0 {
		return throw.IllegalValue()
	}

	if _, err := io.ReadFull(reader, p.PacketSignature[:]); err != nil {
		return err
	}

	digest := hasher.SumToDigest()
	signature := cryptkit.NewSignature(longbits.NewMutableFixedSize(p.PacketSignature[:]), signer.GetSignatureMethod())

	if !signer.IsValidDigestSignature(digest, signature) {
		return throw.IllegalValue()
	}

	return nil
}

// Flags for NodeMessageState
const (
	BodyRqFlag apinetwork.FlagIndex = iota
	RejectListFlag
)

type NodeMsgState struct {
	BodyRq     ParcelId   `insolar-transport:"optional=PacketFlags[0]"` //
	RejectList []ParcelId `insolar-transport:"optional=PacketFlags[1]"` //
	AckList    []ParcelId `insolar-transport:"list=nocount"`            // length is determined by packet size
}

func (p *NodeMsgState) SerializeTo(ctx apinetwork.SerializationContext, writer io.Writer) error {
	packet := NewPacket(NodeMessageState)

	size := uint(0)

	if p.BodyRq != 0 {
		packet.Header.SetFlag(BodyRqFlag, true)
		size += ParcelIdByteSize
	}
	if n := len(p.RejectList); n > 0 {
		packet.Header.SetFlag(RejectListFlag, true)
		size += uint(protokit.SizeVarint64(uint64(n)))
	}
	size += uint(len(p.AckList)) * ParcelIdByteSize
	if size == 0 {
		return throw.IllegalState()
	}

	return packet.SerializeTo(ctx, writer, size, func(writer *iokit.LimitedWriter) error {
		b := make([]byte, writer.RemainingBytes())

		if p.BodyRq != 0 {
			p.BodyRq.PutTo(b)
			b = b[ParcelIdByteSize:]
		}

		if n := len(p.RejectList); n > 0 {
			sz := uint(protokit.EncodeVarintToBytes(b, uint64(n)))
			b = b[sz:]

			for _, id := range p.RejectList {
				if id == 0 {
					return throw.IllegalValue()
				}
				id.PutTo(b)
				b = b[ParcelIdByteSize:]
			}
		}

		for _, id := range p.AckList {
			if id == 0 {
				return throw.IllegalValue()
			}
			id.PutTo(b)
			b = b[ParcelIdByteSize:]
		}
		_, err := writer.Write(b)
		return err
	})
}

func (p *NodeMsgState) DeserializeFrom(ctx apinetwork.DeserializationContext, reader io.Reader) error {
	packet := NewPacket(NodeMessageState)

	return packet.DeserializeFrom(ctx, reader, func(reader *iokit.LimitedReader) error {
		if reader.RemainingBytes() < ParcelIdByteSize {
			return throw.IllegalValue()
		}

		b := make([]byte, reader.RemainingBytes())
		if _, err := reader.Read(b); err != nil {
			return err
		}

		if packet.Header.HasFlag(BodyRqFlag) {
			p.BodyRq = ParcelIdReadFromBytes(b)
			if p.BodyRq == 0 {
				return throw.IllegalValue()
			}
			b = b[ParcelIdByteSize:]
		}

		if packet.Header.HasFlag(RejectListFlag) {
			count, n := protokit.DecodeVarintFromBytes(b)
			if count == 0 || n == 0 {
				return throw.IllegalValue()
			}
			b = b[n:]
			p.RejectList = make([]ParcelId, 0, count)
			for ; count > 0; count-- {
				id := ParcelIdReadFromBytes(b)
				b = b[ParcelIdByteSize:]
				if id == 0 {
					return throw.IllegalValue()
				}
				p.RejectList = append(p.RejectList, id)
			}
		}

		if len(b)%ParcelIdByteSize != 0 {
			return throw.IllegalValue()
		}

		p.AckList = make([]ParcelId, 0, len(b)/ParcelIdByteSize)

		for len(b) > 0 {
			id := ParcelIdReadFromBytes(b)
			if id == 0 {
				return throw.IllegalValue()
			}
			b = b[ParcelIdByteSize:]
			p.AckList = append(p.AckList, id)
		}

		return nil
	})
}

type NodeMsgParcel struct {
	ParcelId ParcelId
	ReturnId ParcelId `insolar-transport:"Packet=1;optional=PacketFlags[0]"`

	RepeatedSend bool                           `insolar-transport:"aliasOf=PacketFlags[1]"`
	ParcelType   apinetwork.PayloadCompleteness `insolar-transport:"send=ignore;aliasOf=PacketFlags[2]"`

	Data apinetwork.Serializable
}

const ( // Flags for NodeMessageParcelX
	ReturnIdFlag apinetwork.FlagIndex = iota
	RepeatedSendFlag
	WithHeadFlag // for NodeMessageParcelBody only
)

func (p *NodeMsgParcel) SerializeTo(ctx apinetwork.SerializationContext, writer io.Writer) error {
	if p.ParcelId == 0 {
		return throw.IllegalState()
	}

	packet := NewPacket(NodeMessageParcelBody)

	switch p.ParcelType {
	case apinetwork.CompletePayload:
		packet.Header.SetFlag(WithHeadFlag, true)
	case apinetwork.BodyPayload:
	case apinetwork.HeadPayload:
		packet.Header.SetPacketType(uint8(NodeMessageParcelHead))
	default:
		return throw.IllegalState()
	}

	size := uint(ParcelIdByteSize)
	if p.ReturnId != 0 {
		size <<= 1
		packet.Header.SetFlag(ReturnIdFlag, true)
	}
	size += p.Data.ByteSize()

	packet.Header.SetFlag(RepeatedSendFlag, p.RepeatedSend)

	return packet.SerializeTo(ctx, writer, size, func(writer *iokit.LimitedWriter) error {
		if err := p.ParcelId.WriteTo(writer); err != nil {
			return err
		}
		if p.ReturnId != 0 {
			if err := p.ReturnId.WriteTo(writer); err != nil {
				return err
			}
		}
		return p.Data.SerializeTo(ctx, writer)
	})
}

func (p *NodeMsgParcel) DeserializeFrom(ctx apinetwork.DeserializationContext, reader io.Reader) error {
	packet := NewPacket(NodeMessageParcelBody) // or NodeMessageParcelHead

	return packet.DeserializeFrom(ctx, reader, func(reader *iokit.LimitedReader) (err error) {
		switch NodeMessageType(packet.Header.GetPacketType()) {
		case NodeMessageParcelHead:
			p.ParcelType = apinetwork.HeadPayload
		case NodeMessageParcelBody:
			if packet.Header.HasFlag(WithHeadFlag) {
				p.ParcelType = apinetwork.CompletePayload
			} else {
				p.ParcelType = apinetwork.BodyPayload
			}
		}

		{
			hasReturn := packet.Header.HasFlag(ReturnIdFlag)
			var b []byte
			if hasReturn {
				b = make([]byte, ParcelIdByteSize<<1)
			} else {
				b = make([]byte, ParcelIdByteSize)
			}
			if _, err := io.ReadFull(reader, b); err != nil {
				return err
			}
			p.ParcelId = ParcelIdReadFromBytes(b)
			if hasReturn {
				p.ReturnId = ParcelIdReadFromBytes(b[ParcelIdByteSize:])
			}
		}

		p.RepeatedSend = packet.Header.HasFlag(RepeatedSendFlag)

		if p.Data != nil {
			return p.Data.DeserializeFrom(ctx, reader)
		}

		factory := ctx.GetPayloadFactory()
		if d, err := factory.DeserializePayloadFrom(ctx, p.ParcelType, reader); err != nil {
			return err
		} else {
			p.Data = d
		}
		return nil
	})
}
