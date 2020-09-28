// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"bytes"
	"io"
	"net"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

// ReceivedPacket represents a received packet with additional information,
// Also ReceivedPacket provides a few convenience functions to unify deserialization of different packet types.
type ReceivedPacket struct {
	// Packet is a standard part of a packet.
	Packet
	// From is a transport-specific address of a remote endpoint.
	From      nwapi.Address
	// Peer represents a peer for the given remote endpoint.
	Peer      Peer
	verifier  PacketVerifier
	// Decrypter is provided when packet's content is encrypted.
	Decrypter cryptkit.Decrypter
}

// PayloadDeserializeFunc should read packet and its content. Param (iokit.LimitedReader) contains data past the packet's default header.
type PayloadDeserializeFunc func(nwapi.DeserializationContext, *Packet, *iokit.LimitedReader) error
type PacketDeserializerFunc func(nwapi.DeserializationContext, PayloadDeserializeFunc) error

// NewSmallPayloadDeserializer returns a unified PacketDeserializerFunc to read a small (fully-read) packet.
// Param (b) is a fully read packet, and it MUST correlate with p.Packet.
func (p *ReceivedPacket) NewSmallPayloadDeserializer(b []byte) PacketDeserializerFunc {
	payload := b[p.GetPayloadOffset() : len(b)-p.verifier.GetSignatureSize()]
	reader := bytes.NewReader(payload)

	return func(ctx nwapi.DeserializationContext, fn PayloadDeserializeFunc) error {
		return p.DeserializePayload(ctx, reader, reader.Size(), p.Decrypter, fn)
	}
}

// NewLargePayloadDeserializer returns a unified PacketDeserializerFunc to a large (partially-read) packet.
// Param (preRead) is a pre-read portion of the packet, and it MUST correlate with p.Packet.
// Param (r) is unread portion of the packet from the transport.
// WARNING! Transfer of other large packets of all protocols is BLOCKED until content of (r) will be read.
func (p *ReceivedPacket) NewLargePayloadDeserializer(preRead []byte, r io.LimitedReader) PacketDeserializerFunc {

	skip, hasher := p.verifier.NewHasher(&p.Header)
	sigSize := p.verifier.GetSignatureSize()
	if sigSize > 0 {
		r.N -= int64(sigSize)
	}

	total := r.N
	var reader io.Reader
	switch ofs := p.GetPayloadOffset() + uint(sigSize); {
	case int(ofs) == len(preRead):
		hasher.DigestBytes(preRead[skip:])
		reader = iokit.NewTeeReader(&r, hasher)

	case int(ofs) < len(preRead):
		hasher.DigestBytes(preRead[skip:])
		reader = iokit.NewTeeReader(&r, hasher)
		preRead = preRead[ofs:]
		total += int64(len(preRead))
		reader = iokit.PrependReader(preRead, reader)

	case len(preRead) == 0:
		reader = iokit.NewTeeReaderWithSkip(&r, hasher, skip)

	default:
		panic(throw.Impossible())
	}

	return func(ctx nwapi.DeserializationContext, fn PayloadDeserializeFunc) error {
		if err := p.DeserializePayload(ctx, reader, total, p.Decrypter, fn); err != nil {
			return err
		}
		r.N += int64(sigSize)
		b := make([]byte, sigSize)
		digest := hasher.SumToDigest()
		n, err := io.ReadFull(&r, b)
		if n == sigSize {
			err = p.verifier.VerifySignature(digest, b)
		}
		return err
	}
}

// DownstreamError filters errors from downstream (from transport) as they will be handled there.
// DO NOT attempt to remediate transport errors.
func (ReceivedPacket) DownstreamError(err error) error {
	switch err {
	case io.EOF, io.ErrUnexpectedEOF:
		return err
	}
	if _, ok := err.(net.Error); ok {
		return err
	}
	if sv := throw.SeverityOf(err); sv.IsFraudOrWorse() {
		return throw.Severe(sv, "aborted")
	}
	return nil
}

// GetSignatureSize returns size of a signature applicable fot this packet.
func (p *ReceivedPacket) GetSignatureSize() int {
	return p.verifier.GetSignatureSize()
}

// GetContext is a helper to create nwapi.DeserializationContext with necessary dependencies.
func (p *ReceivedPacket) GetContext(factory nwapi.DeserializationFactory) nwapi.DeserializationContext {
	return packetContext{factory}
}
