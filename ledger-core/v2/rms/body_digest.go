// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ GoGoSerializable = &BodyWithDigest{}
var _ DigestProvider = &BodyWithDigest{}

type BodyWithDigest struct {
	data   Serializable
	digest digestProvider
}

func (p *BodyWithDigest) GetDigestMethod() cryptkit.DigestMethod {
	return p.digest.GetDigestMethod()
}

func (p *BodyWithDigest) GetDigestSize() int {
	return p.digest.GetDigestSize()
}

func (p *BodyWithDigest) GetDigest() cryptkit.Digest {
	return p.digest.GetDigest()
}

func (p *BodyWithDigest) _digestData(digester cryptkit.DataDigester) cryptkit.Digest {
	switch {
	case digester == nil:
		if p.data == nil {
			return cryptkit.NewZeroSizeDigest("")
		}
		panic(throw.IllegalState())
	case p.data == nil:
		return digester.DigestBytes(nil)
	}
	data, err := p.data.Marshal()
	if err != nil {
		panic(throw.WithStackTop(err))
	}
	return digester.DigestBytes(data)
}

func (p *BodyWithDigest) ProtoSize() int {
	p.digest.calcDigest(p._digestData, nil)
	return protokit.BinaryProtoSize(p.digestSize())
}

func (p *BodyWithDigest) digestSize() int {
	return p.digest.digest.FixedByteSize()
}

func (p *BodyWithDigest) mustDigestSize() int {
	if !p.digest.isReady() {
		panic(throw.IllegalState())
	}
	return p.digestSize()
}

func (p *BodyWithDigest) MarshalTo(b []byte) (int, error) {
	return protokit.BinaryMarshalTo(b, func(b []byte) (int, error) {
		switch n := p.mustDigestSize(); {
		case n == 0:
			return 0, nil
		case n > len(b):
			return 0, io.ErrShortBuffer
		}
		return p.digest.digest.CopyTo(b), nil
	})
}

func (p *BodyWithDigest) MarshalToSizedBuffer(b []byte) (int, error) {
	return protokit.BinaryMarshalToSizedBuffer(b, func(b []byte) (int, error) {
		switch n := p.mustDigestSize(); {
		case n == 0:
			return 0, nil
		case n > len(b):
			return 0, io.ErrShortBuffer
		default:
			i := len(b) - n
			p.digest.digest.CopyTo(b[i:])
			return n, nil
		}
	})
}

func (p *BodyWithDigest) Unmarshal(b []byte) error {
	return protokit.BinaryUnmarshal(b, func(b []byte) error {
		digest := cryptkit.NewZeroSizeDigest("")
		if len(b) > 0 {
			digest = cryptkit.NewDigest(longbits.NewImmutableFixedSize(b[1:]), "")
		}
		p.digest.setDigest(digest, nil)
		return nil
	})
}

func (p *BodyWithDigest) SetPayload(data Serializable, digester cryptkit.DataDigester) {
	p.digest.setDigester(digester, func() {
		p.data = data
	})
}

func (p *BodyWithDigest) GetPayload() Serializable {
	if p.digest.ready.WasStarted() {
		return p.data
	}
	return nil
}

func (p *BodyWithDigest) VerifyPayloadBytes(data []byte, digester cryptkit.DataDigester) error {
	d0 := p.digest.GetDigest()
	var d1 cryptkit.Digest
	switch {
	case !d0.IsEmpty():
		d1 := digester.DigestBytes(data)
		if m0 := d0.GetDigestMethod(); m0 != "" && m0 == d1.GetDigestMethod() {
			return throw.E("digest method mismatched", struct{ M0, M1 cryptkit.DigestMethod }{m0, d1.GetDigestMethod()})
		}
		if longbits.Equal(d0, d1) {
			return nil
		}
	case len(data) == 0:
		return nil
	}
	return throw.E("digest mismatched", struct{ D0, D1 cryptkit.Digest }{d0, d1})
}

func (p *BodyWithDigest) VerifyPayload(data Serializable, digester cryptkit.DataDigester) error {
	if data == nil {
		return p.VerifyPayloadBytes(nil, digester)
	}
	b, err := data.Marshal()
	if err != nil {
		return err
	}
	return p.VerifyPayloadBytes(b, digester)
}

func (p *BodyWithDigest) Equal(o *BodyWithDigest) bool {
	switch {
	case p == o:
		return true
	case o == nil || p == nil:
		return false
	}
	return longbits.Equal(p.GetDigest(), o.GetDigest())
}
