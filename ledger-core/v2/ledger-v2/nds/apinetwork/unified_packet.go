// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package apinetwork

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

const (
	PacketByteSizeMin                       = HeaderByteSizeMin + pulse.NumberSize
	LargePacketBaselineWithoutSignatureSize = HeaderByteSizeMax + pulse.NumberSize //+ PacketSignatureSize
)

type Packet struct {
	Header Header

	PulseNumber pulse.Number `insolar-transport:"[30-31]=0"` // depends on protocol

	// SourceKey []byte // depends on protocol

	// HeaderSignature provides earlier verification for large packets to prevent
	// an unauthorized sender from sending large data packets
	HeaderSignature []byte   `insolar-transport:"optional=IsExcessiveLength"`
	EncryptableBody struct{} `insolar-transport:"send=placeholder"`
	EncryptionData  []byte   `insolar-transport:"optional=IsBodyEncrypted"`
	PacketSignature []byte   `insolar-transport:"generate=signature"` // can be zero length, depends on protocol
}

func (p *Packet) SerializeTo(ctx SerializationContext, writer io.Writer, dataSize uint, fn func(*iokit.LimitedWriter) error) error {
	if pn, err := ctx.PrepareHeader(&p.Header, p.PulseNumber); err != nil {
		return err
	} else {
		p.PulseNumber = pn
	}

	signer := ctx.GetPayloadSigner()
	hasher := signer.NewHasher()
	var encrypter cryptkit.Encrypter

	payloadSize := dataSize

	if p.Header.IsBodyEncrypted() {
		encrypter = ctx.GetPayloadEncrypter()
		payloadSize += encrypter.GetOverheadSize(payloadSize)
	}

	signatureSize := uint(signer.GetDigestSize())

	payloadSize += uint(pulse.NumberSize)
	payloadSize += signatureSize // PacketSignature

	packetSize := p.Header.SetPayloadLength(uint64(payloadSize))
	if p.Header.IsExcessiveLength() {
		payloadSize += signatureSize // HeaderSignature
		packetSize = p.Header.SetPayloadLength(uint64(payloadSize))
	}

	if err := ctx.VerifyHeader(&p.Header, p.PulseNumber); err != nil {
		return err
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

	if encrypter != nil {
		encWriter := encrypter.NewEncryptingWriter(teeWriter, dataSize)
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

func (p *Packet) VerifyExcessivePayload(signer cryptkit.DataSignatureVerifier, preload *[]byte, r io.Reader) error {
	const x = LargePacketBaselineWithoutSignatureSize
	n := signer.GetDigestSize()
	requiredLen := x + n
	b := *preload
	if l := len(b); requiredLen > l {
		b = append(b, make([]byte, requiredLen-l)...)

		if _, err := io.ReadFull(r, b[l:]); err != nil {
			return err
		}

		*preload = b
	}

	hasher := signer.NewHasher()
	zeroPrefix := p.Header.GetHashingZeroPrefix()

	_, _ = iokit.WriteZeros(zeroPrefix, hasher)
	_, _ = hasher.Write(b[zeroPrefix:x])
	digest := hasher.SumToDigest()
	return p.verifySignature(signer, digest, b[x:requiredLen])
}

func (p *Packet) VerifyNonExcessivePayload(signer cryptkit.DataSignatureVerifier, b []byte) error {
	n := signer.GetDigestSize()
	switch limit, err := p.Header.GetFullLength(); {
	case err != nil:
		return err
	case limit != uint64(len(b)):
		return throw.IllegalValue()
	case limit < uint64(PacketByteSizeMin+n):
		return throw.IllegalValue()
	}

	hasher := signer.NewHasher()
	zeroPrefix := p.Header.GetHashingZeroPrefix()
	_, _ = iokit.WriteZeros(zeroPrefix, hasher)
	_, _ = hasher.Write(b[zeroPrefix : len(b)-n])
	digest := hasher.SumToDigest()
	return p.verifySignature(signer, digest, b[len(b)-n:])
}

func (p *Packet) DeserializeFrom(ctx DeserializationContext, reader io.Reader, fn func(*iokit.LimitedReader) error) error {
	signer := ctx.GetPayloadVerifier()
	hasher := signer.NewHasher()

	teeReader := iokit.NewTeeReaderWithSkip(reader, hasher, p.Header.GetHashingZeroPrefix())
	_, _ = iokit.WriteZeros(p.Header.GetHashingZeroPrefix(), hasher)

	if err := p.DeserializeMinFrom(teeReader); err != nil {
		return err
	}

	readLimit := int64(0)
	if limit, err := p.Header.GetPayloadLength(); err != nil {
		return err
	} else {
		readLimit = int64(limit)
	}

	switch n := int64(signer.GetDigestSize()); {
	case readLimit < n:
		return throw.IllegalValue()
	case !p.Header.IsExcessiveLength():
		readLimit -= n
	case readLimit < n<<1:
		return throw.IllegalValue()
	default:
		readLimit -= n << 1

		digest := hasher.SumToDigest() // must be taken before reading the signature
		p.HeaderSignature = make([]byte, n)
		if _, err := io.ReadFull(reader, p.HeaderSignature); err != nil {
			return err
		} else if err = p.verifySignature(signer, digest, p.HeaderSignature); err != nil {
			return err
		}
	}

	if err := ctx.VerifyHeader(&p.Header, p.PulseNumber); err != nil {
		return err
	}

	if readLimit > 0 && p.Header.IsBodyEncrypted() {
		decrypter := ctx.GetPayloadDecrypter()

		encReader, plainSize := decrypter.NewDecryptingReader(teeReader, uint(readLimit))
		limitReader := iokit.LimitReader(encReader, int64(plainSize))

		if err := fn(limitReader); err != nil {
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
	} else {
		limitReader := iokit.LimitReader(teeReader, readLimit)
		if err := fn(limitReader); err != nil {
			return err
		}
		if limitReader.RemainingBytes() != 0 {
			return throw.IllegalValue()
		}
	}

	digest := hasher.SumToDigest()
	p.PacketSignature = make([]byte, signer.GetDigestSize())
	if _, err := io.ReadFull(reader, p.PacketSignature); err != nil {
		return err
	}
	return p.verifySignature(signer, digest, p.PacketSignature)
}

func (p *Packet) verifySignature(verifier cryptkit.DataSignatureVerifier, digest cryptkit.Digest, signatureBytes []byte) error {
	signature := cryptkit.NewSignature(longbits.NewMutableFixedSize(signatureBytes), verifier.GetSignatureMethod())

	if !verifier.IsValidDigestSignature(digest, signature) {
		return throw.IllegalValue()
	}
	return nil
}

func (p *Packet) GetPayloadOffset() uint {
	return p.Header.ByteSize() + pulse.NumberSize
}
