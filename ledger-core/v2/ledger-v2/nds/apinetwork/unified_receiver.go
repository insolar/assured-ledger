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
	From       Address
	verifier   PacketDataVerifier
	decFactory func() cryptkit.Decrypter
}

type PacketDeserializerFunc func(PayloadDeserializeFunc) error

func (p ReceiverPacket) NewSmallPayloadDeserializer(b []byte) PacketDeserializerFunc {
	payload := b[p.GetPayloadOffset() : len(b)-p.verifier.GetSignatureSize()]
	reader := bytes.NewReader(payload)

	return func(fn PayloadDeserializeFunc) error {
		return p.DeserializePayload(reader, reader.Size(), p.decFactory, fn)
	}
}

func (p ReceiverPacket) NewLargePayloadDeserializer(preRead []byte, r io.LimitedReader) PacketDeserializerFunc {
	if ofs := p.GetPayloadOffset(); int(ofs) >= len(preRead) {
		preRead = nil
	} else {
		preRead = preRead[ofs:]
	}
	if n := p.verifier.GetSignatureSize(); n > 0 {
		r.N -= int64(n)
	}
	total := int64(len(preRead)) + r.N
	reader := iokit.PrependReader(preRead, &r)

	return func(fn PayloadDeserializeFunc) error {
		return p.DeserializePayload(reader, total, p.decFactory, fn)
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

func (p ReceiverPacket) GetSignatureSize() int {
	return p.verifier.GetSignatureSize()
}
