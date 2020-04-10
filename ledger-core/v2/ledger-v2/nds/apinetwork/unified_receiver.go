// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package apinetwork

import (
	"bytes"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type ReceiverPacket struct {
	Packet
	From      Address
	verifier  PacketDataVerifier
	Decrypter cryptkit.Decrypter
}

type PacketDeserializerFunc func(PayloadDeserializeFunc) error

func (p *ReceiverPacket) NewSmallPayloadDeserializer(b []byte) PacketDeserializerFunc {
	payload := b[p.GetPayloadOffset() : len(b)-p.verifier.GetSignatureSize()]
	reader := bytes.NewReader(payload)

	return func(fn PayloadDeserializeFunc) error {
		return p.DeserializePayload(reader, reader.Size(), p.Decrypter, fn)
	}
}

func (p *ReceiverPacket) NewLargePayloadDeserializer(preRead []byte, r io.LimitedReader) PacketDeserializerFunc {

	skip, hasher := p.verifier.NewHasher(&p.Header)

	//if skip >= len(preRead) {
	//	tr := cryptkit.NewHashingTeeReader(hasher, r)
	//	tr.CopySkip = skip - len(preRead)
	//	return &tr
	//}
	//
	//if len(preRead) > skip {
	//	hasher.DigestBytes(preRead[skip:])
	//	tr := cryptkit.NewHashingTeeReader(hasher, iokit.PrependReader())
	//} else {
	//	skip -= len(preRead)
	//}

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

	return func(fn PayloadDeserializeFunc) error {
		if err := p.DeserializePayload(reader, total, p.Decrypter, fn); err != nil {
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

func (ReceiverPacket) DownstreamError(err error) error {
	if err == io.EOF {
		return err
	}
	if sv := throw.SeverityOf(err); sv.IsFraudOrWorse() {
		return throw.Severe(sv, "aborted")
	}
	return nil
}

func (p *ReceiverPacket) GetSignatureSize() int {
	return p.verifier.GetSignatureSize()
}
