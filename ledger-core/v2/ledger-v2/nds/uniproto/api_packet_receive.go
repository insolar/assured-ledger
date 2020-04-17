// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"bytes"
	"io"
	"net"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type ReceivedPacket struct {
	Packet
	From      nwapi.Address
	Peer      Peer
	verifier  PacketVerifier
	Decrypter cryptkit.Decrypter
}

type PayloadDeserializeFunc func(nwapi.DeserializationContext, *Packet, *iokit.LimitedReader) error
type PacketDeserializerFunc func(nwapi.DeserializationContext, PayloadDeserializeFunc) error

func (p *ReceivedPacket) NewSmallPayloadDeserializer(b []byte) PacketDeserializerFunc {
	payload := b[p.GetPayloadOffset() : len(b)-p.verifier.GetSignatureSize()]
	reader := bytes.NewReader(payload)

	return func(ctx nwapi.DeserializationContext, fn PayloadDeserializeFunc) error {
		return p.DeserializePayload(ctx, reader, reader.Size(), p.Decrypter, fn)
	}
}

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

func (p *ReceivedPacket) GetSignatureSize() int {
	return p.verifier.GetSignatureSize()
}

func (p *ReceivedPacket) GetContext(factory nwapi.DeserializationFactory) nwapi.DeserializationContext {
	return packetContext{factory}
}
