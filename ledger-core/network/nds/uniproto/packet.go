// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniproto

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
)

const (
	PacketByteSizeMin                       = HeaderByteSizeMin + pulse.NumberSize
	LargePacketBaselineWithoutSignatureSize = HeaderByteSizeMax + pulse.NumberSize // + PacketSignatureSize
)

// Packet represents a logical packet of a protocol
type Packet struct {
	Header      Header
	PulseNumber pulse.Number `insolar-transport:"[30-31]=0"`

	// The fields below are optional, their presence depends on packet type and size.
	/*

	SourcePK []byte // self-identified packets, not implemented, presence depends on protocol

	// HeaderSignature provides earlier verification for large packets to prevent
	// an unauthorized sender from sending large data packets
	HeaderSignature []byte   `insolar-transport:"optional=IsExcessiveLength"`

	EncryptableBody struct{} `insolar-transport:"send=placeholder"`
	EncryptionData  []byte   `insolar-transport:"optional=IsBodyEncrypted"`

	PacketSignature []byte   `insolar-transport:"generate=signature"` // can be zero length, depends on protocol

	*/
}

func (p *Packet) SerializeMinToBytes(b []byte) uint {
	n := p.Header.SerializeToBytes(b)
	SerializePulseNumberToBytes(p.PulseNumber, b[n:])
	return n + pulse.NumberSize
}

func (p *Packet) SerializePayload(ctx nwapi.SerializationContext, writer *iokit.LimitedWriter, dataSize uint, encrypter cryptkit.Encrypter, fn PayloadSerializerFunc) error {
	if !p.Header.IsBodyEncrypted() {
		return fn(ctx, p, writer)
	}

	encWriter := encrypter.NewEncryptingWriter(writer, dataSize)
	teeEncWriter := iokit.LimitWriter(encWriter, int64(dataSize))
	if err := fn(ctx, p, teeEncWriter); err != nil {
		return err
	}
	if cw, ok := encWriter.(io.Closer); ok {
		// enables use of AEAD-like encryption with extra data on closing
		if err := cw.Close(); err != nil {
			panic(throw.W(err, "AEAD finalization"))
		}
	}
	return nil
}

func (p *Packet) DeserializeMinFrom(reader io.Reader) error {
	if err := p.Header.DeserializeFrom(reader); err != nil {
		return err
	}

	pn, err := DeserializePulseNumber(reader)
	if err != nil {
		return err
	}
	p.PulseNumber = pn

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

func (p *Packet) GetPacketSize(dataSize, sigSize uint) uint {
	dataSize += pulse.NumberSize
	return p.Header.GetPacketSize(dataSize, sigSize)
}
