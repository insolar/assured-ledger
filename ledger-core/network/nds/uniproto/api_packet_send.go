// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"bytes"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nds/uniproto/l1"
	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

// PacketPreparer is a factory to provide a to-be-sent packet and necessary information.
type PacketPreparer interface {
	// PreparePacket returns pre-populated header, size of data to be included into the packet (excludes all packet fields),
	// and a data serialization function.
	PreparePacket() (template PacketTemplate, dataSize uint, dataFn PayloadSerializerFunc)
}

type ProtocolPacket interface {
	PacketPreparer
	SerializePayload(nwapi.SerializationContext, *Packet, *iokit.LimitedWriter) error
	DeserializePayload(nwapi.DeserializationContext, *Packet, *iokit.LimitedReader) error
}

type PacketTemplate struct {
	Packet
}

type PayloadSerializerFunc func(nwapi.SerializationContext, *Packet, *iokit.LimitedWriter) error
type PacketSerializerFunc func() (template PacketTemplate, dataSize uint, dataFn PayloadSerializerFunc)

// NewSendingPacket creates a to-be-sent packet. Param (encrypter) is only required when encryption is needed.
func NewSendingPacket(signer cryptkit.DataSigner, encrypter cryptkit.Encrypter) *SendingPacket {
	p := &SendingPacket{encrypter: encrypter}
	p.signer.Signer = signer
	return p
}

// ReceivedPacket represents a to-be-sent packet.
// Also ReceivedPacket provides a few convenience functions to unify serialization of different packet types.
type SendingPacket struct {
	// Packet is a standard part of a packet.
	Packet
	// Peer represents a peer for this packet. Not in use.
	Peer      Peer
	signer    PacketDataSigner
	encrypter cryptkit.Encrypter
}

func (p *SendingPacket) GetContext() nwapi.SerializationContext {
	return nil // TODO GetContext() nwapi.SerializationContext
}

func (p *SendingPacket) preparePacketSize(dataSize uint) uint {
	payloadSize := dataSize

	if p.Header.IsBodyEncrypted() {
		payloadSize += p.encrypter.GetOverheadSize(payloadSize)
	}

	signatureSize := p.signer.GetSignatureSize()

	payloadSize += uint(pulse.NumberSize)
	payloadSize += signatureSize // PacketSignature

	packetSize := p.Header.SetPayloadLength(uint64(payloadSize))
	if p.Header.IsExcessiveLength() {
		payloadSize += signatureSize // HeaderSignature
		packetSize = p.Header.SetPayloadLength(uint64(payloadSize))
	}

	return uint(packetSize)
}

func (p *SendingPacket) preSerialize(dataSize uint, alwaysComplete bool) ([]byte, uint, uint) {
	if !pulse.IsValidAsPulseNumber(int(p.PulseNumber)) {
		panic(throw.IllegalState())
	}

	packetSize := p.preparePacketSize(dataSize)

	preBufSize := packetSize
	if !alwaysComplete && packetSize > MaxNonExcessiveLength {
		preBufSize = LargePacketBaselineWithoutSignatureSize + p.signer.GetSignatureSize()
	}
	preBuf := make([]byte, preBufSize)
	n := p.Packet.SerializeMinToBytes(preBuf)

	return preBuf, n, packetSize
}

// SerializeToBytes assists with serialization of a packet into memory from the given payload data size
// and PayloadSerializerFunc
func (p *SendingPacket) SerializeToBytes(dataSize uint, fn PayloadSerializerFunc) ([]byte, error) {
	preBuf, n, packetSize := p.preSerialize(dataSize, true)
	hasher := p.signer.NewHasherWith(&p.Header, preBuf[:n])

	sigSize := p.GetSignatureSize()
	payloadSize := packetSize - n - sigSize

	if packetSize > MaxNonExcessiveLength {
		nn := n + sigSize
		p.signer.SumToSignatureBytes(hasher, preBuf[n:nn])
		hasher.DigestBytes(preBuf[n:nn])
		n = nn
		payloadSize -= sigSize
	}

	p.serializeToBytes(preBuf[:n], dataSize, payloadSize, fn, hasher)
	return preBuf, nil
}

// NewTransportFunc provides ready-to-use arguments for OutTransport methods.
// for the given payload data and PayloadSerializerFunc.
// When option (checkFn) is provided, it will be invoked before each send attempt and it can return false to prevent from (re)sending.
func (p *SendingPacket) NewTransportFunc(dataSize uint, fn PayloadSerializerFunc, checkFn func() bool) (uint, OutFunc) {
	preBuf, n, packetSize := p.preSerialize(dataSize, false)

	hasher := p.signer.NewHasherWith(&p.Header, preBuf[:n])

	if packetSize <= MaxNonExcessiveLength {
		payloadSize := packetSize - n - p.GetSignatureSize()

		isRepeated := false
		return packetSize, func(t l1.BasicOutTransport) (canRetry bool, err error) {
			if checkFn != nil && !checkFn() {
				return false, nil
			}
			if !isRepeated {
				p.serializeToBytes(preBuf[:n], dataSize, payloadSize, fn, hasher)
				isRepeated = true
			}
			return true, t.SendBytes(preBuf)
		}
	}

	p.signer.SumToSignatureBytes(hasher, preBuf[n:])
	hasher.DigestBytes(preBuf[n:])
	isRepeated := false

	payloadSize := packetSize - uint(len(preBuf)) - p.GetSignatureSize()

	return packetSize, func(t l1.BasicOutTransport) (bool, error) {
		if checkFn != nil && !checkFn() {
			return false, nil
		}

		if err := t.SendBytes(preBuf); err != nil {
			return true, err
		}
		if !isRepeated {
			isRepeated = true
		} else {
			// this is to enable repeatable send
			hasher = p.signer.NewHasherWith(&p.Header, preBuf)
		}

		pw := packetWriterTo{p: p, dSize: dataSize, pSize: payloadSize, fn: fn, hash: hasher}
		switch err := t.Send(&pw); {
		case err == nil:
			//
		case pw.tErr.e != nil:
			return true, err
		default:
			panic(throw.W(err, "serializer error"))
		}
		err := t.Send(hasher.SumToSignature(p.signer.Signer))
		return err != nil, err
	}
}

// GetSignatureSize returns size of a signature applicable fot this packet.
func (p *SendingPacket) GetSignatureSize() uint {
	return p.signer.GetSignatureSize()
}

func (p *SendingPacket) serializeToBytes(b []byte, dataSize, payloadSize uint, fn PayloadSerializerFunc, hasher cryptkit.DigestHasher) {
	buf := bytes.NewBuffer(b)
	n, err := p.writePayload(buf, dataSize, payloadSize, fn, hasher)
	if err != nil {
		panic(throw.W(err, "unexpected"))
	}
	p.signer.SumToSignatureBytes(hasher, b[int64(len(b))+n:cap(b)])
}

func (p *SendingPacket) writePayload(w io.Writer, dataSize, payloadSize uint, fn PayloadSerializerFunc, hasher cryptkit.DigestHasher) (int64, error) {
	writer := iokit.NewLimitedTeeWriter(w, hasher, int64(payloadSize))

	err := p.Packet.SerializePayload(p.GetContext(), writer, dataSize, p.encrypter, fn)
	if err == nil && writer.RemainingBytes() != 0 {
		panic(throw.FailHere("size mismatch"))
	}
	return int64(payloadSize) - writer.RemainingBytes(), err
}

/********************************/

type PacketDataSigner struct {
	Signer cryptkit.DataSigner
}

func (v PacketDataSigner) GetSignatureSize() uint {
	return uint(v.Signer.GetDigestSize())
}

func (v PacketDataSigner) NewHasher(h *Header) (int, cryptkit.DigestHasher) {
	zeroPrefixLen := h.GetHashingZeroPrefix()
	hasher := v.Signer.NewHasher()
	_, _ = iokit.WriteZeros(zeroPrefixLen, hasher)
	return zeroPrefixLen, hasher
}

func (v PacketDataSigner) NewHasherWith(h *Header, b []byte) cryptkit.DigestHasher {
	zeroPrefixLen, hasher := v.NewHasher(h)
	hasher.DigestBytes(b[zeroPrefixLen:])
	return hasher
}

func (v PacketDataSigner) SumToSignatureBytes(hasher cryptkit.DigestHasher, b []byte) {
	signature := hasher.SumToSignature(v.Signer)
	switch sn := signature.CopyTo(b); {
	case sn != signature.FixedByteSize():
		panic(throw.IllegalValue())
	case len(b) != sn:
		panic(throw.IllegalValue())
	}
}

/********************************/

type packetWriterTo struct {
	p     *SendingPacket
	dSize uint
	pSize uint
	fn    PayloadSerializerFunc
	hash  cryptkit.DigestHasher
	tErr  errorCatcher
}

func (p *packetWriterTo) WriteTo(w io.Writer) (int64, error) {
	p.tErr.e = nil
	p.tErr.w = w
	return p.p.writePayload(&p.tErr, p.dSize, p.pSize, p.fn, p.hash)
}

type errorCatcher struct {
	w io.Writer
	e error
}

func (p *errorCatcher) Write(b []byte) (n int, err error) {
	n, err = p.w.Write(b)
	p.e = err
	return
}
