// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/ledger-v2/nds/nwapi"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

const (
	PacketByteSizeMin                       = HeaderByteSizeMin + pulse.NumberSize
	LargePacketBaselineWithoutSignatureSize = HeaderByteSizeMax + pulse.NumberSize //+ PacketSignatureSize
)

type Packet struct {
	Header      Header
	PulseNumber pulse.Number `insolar-transport:"[30-31]=0"`

	/*

		SourceKey []byte // self-identified packets, not implemented, presence depends on protocol

		// HeaderSignature provides earlier verification for large packets to prevent
		// an unauthorized sender from sending large data packets
		HeaderSignature []byte   `insolar-transport:"optional=IsExcessiveLength"`
		EncryptableBody struct{} `insolar-transport:"send=placeholder"`
		EncryptionData  []byte   `insolar-transport:"optional=IsBodyEncrypted"`
		PacketSignature []byte   `insolar-transport:"generate=signature"` // can be zero length, depends on protocol

	*/
}

func (p *Packet) SerializePayload(packet *SendingPacket, writer io.Writer, dataSize uint, fn func(*iokit.LimitedWriter) error) error {
	if !pulse.IsValidAsPulseNumber(int(packet.PulseNumber)) {
		return throw.E("invalid pulse")
	}

	signer := packet.signer.signer
	hasher := signer.NewHasher()
	payloadSize := dataSize

	if p.Header.IsBodyEncrypted() {
		payloadSize += packet.encrypter.GetOverheadSize(payloadSize)
	}

	signatureSize := uint(signer.GetDigestSize())

	payloadSize += uint(pulse.NumberSize)
	payloadSize += signatureSize // PacketSignature

	packetSize := p.Header.SetPayloadLength(uint64(payloadSize))
	if p.Header.IsExcessiveLength() {
		payloadSize += signatureSize // HeaderSignature
		packetSize = p.Header.SetPayloadLength(uint64(payloadSize))
	}

	if packet.packetSizeLimit > 0 && packetSize > packet.packetSizeLimit {
		return throw.E("packet size limit exceeded")
	}

	teeWriter := iokit.NewLimitedTeeWriterWithSkip(writer, hasher, p.Header.GetHashingZeroPrefix(),
		int64(packetSize-uint64(signatureSize)))

	_, _ = iokit.WriteZeros(p.Header.GetHashingZeroPrefix(), hasher)

	if err := p.Header.SerializeTo(teeWriter); err != nil {
		return err
	}

	if err := SerializePulseNumber(p.PulseNumber, teeWriter); err != nil {
		return err
	}

	if p.Header.IsExcessiveLength() {
		digest := hasher.SumToDigest()
		signature := signer.SignDigest(digest)

		switch n, err := signature.WriteTo(teeWriter); {
		case err != nil:
			return err
		case n != int64(signature.FixedByteSize()):
			return io.ErrShortWrite
		}
	}

	if p.Header.IsBodyEncrypted() {
		encWriter := packet.encrypter.NewEncryptingWriter(teeWriter, dataSize)
		teeEncWriter := iokit.LimitWriter(encWriter, int64(dataSize))

		if err := fn(teeEncWriter); err != nil {
			return err
		}
		if cw, ok := encWriter.(io.Closer); ok {
			// enables use of AEAD-like encryption with extra data on closing
			if err := cw.Close(); err != nil {
				return err
			}
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

	switch n, err := signature.WriteTo(writer); {
	case err != nil:
		return err
	case n != int64(signature.FixedByteSize()):
		return io.ErrShortWrite
	}
	return nil
}

func (p *Packet) DeserializeMinFrom(reader io.Reader) error {
	if err := p.Header.DeserializeFrom(reader); err != nil {
		return err
	}

	if pn, err := DeserializePulseNumber(reader); err != nil {
		return err
	} else {
		p.PulseNumber = pn
	}

	return nil
}

func (p *Packet) DeserializeMinFromBytes(b []byte) (int, error) {
	if err := p.Header.DeserializeMinFromBytes(b); err != nil {
		return 0, err
	}
	if n, err := p.Header.DeserializeRestFromBytes(b); err != nil {
		return 0, err
	} else if pn, err := DeserializePulseNumberFromBytes(b[n:]); err != nil {
		return 0, err
	} else {
		p.PulseNumber = pn
		return n + pulse.NumberSize, nil
	}
}

func (p *Packet) VerifyExcessivePayload(sv PacketVerifier, preload *[]byte, r io.Reader) error {
	const x = LargePacketBaselineWithoutSignatureSize
	n := sv.GetSignatureSize()
	requiredLen := x + n
	b := *preload
	if l := len(b); requiredLen > l {
		b = append(b, make([]byte, requiredLen-l)...)

		if _, err := io.ReadFull(r, b[l:]); err != nil {
			return err
		}
		*preload = b
	}

	return sv.VerifyWhole(&p.Header, b[:requiredLen])
}

func (p *Packet) VerifyNonExcessivePayload(sv PacketVerifier, b []byte) error {
	switch limit, err := p.Header.GetFullLength(); {
	case err != nil:
		return err
	case limit != uint64(len(b)):
		return throw.IllegalValue()
	}

	return sv.VerifyWhole(&p.Header, b)
}

type PayloadDeserializeFunc func(nwapi.DeserializationContext, *Packet, *iokit.LimitedReader) error

func (p *Packet) DeserializePayload(ctx nwapi.DeserializationContext, r io.Reader, readLimit int64, decrypter cryptkit.Decrypter, fn PayloadDeserializeFunc) error {
	if readLimit == 0 || !p.Header.IsBodyEncrypted() {
		limitReader := iokit.LimitReader(r, readLimit)
		if err := fn(ctx, p, limitReader); err != nil {
			return err
		}
		if limitReader.RemainingBytes() != 0 {
			return throw.IllegalValue()
		}
		return nil
	}

	encReader, plainSize := decrypter.NewDecryptingReader(r, uint(readLimit))
	limitReader := iokit.LimitReader(encReader, int64(plainSize))

	if err := fn(ctx, p, limitReader); err != nil {
		return err
	}
	if cr, ok := encReader.(io.Closer); ok {
		// enables use of AEAD-like encryption with extra data/validation on closing
		if err := cr.Close(); err != nil {
			return err
		}
	}

	if limitReader.RemainingBytes() != 0 {
		return throw.IllegalValue()
	}
	return nil
}

func (p *Packet) GetPayloadOffset() uint {
	return p.Header.ByteSize() + pulse.NumberSize
}
