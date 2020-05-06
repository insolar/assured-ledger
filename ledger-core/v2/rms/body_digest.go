// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ DigestProvider = &RawWithDigest{}

type RawWithDigest struct {
	data   RawBinary
	digest digestProvider
}

func (p *RawWithDigest) GetDigestMethod() cryptkit.DigestMethod {
	return p.digest.GetDigestMethod()
}

func (p *RawWithDigest) GetDigestSize() int {
	return p.digest.GetDigestSize()
}

func (p *RawWithDigest) GetDigest() cryptkit.Digest {
	return p.digest.GetDigest()
}

func (p *RawWithDigest) _digestData(digester cryptkit.DataDigester) cryptkit.Digest {
	return digestBy(p.data, digester)
}

func digestBy(data RawBinary, digester cryptkit.DataDigester) cryptkit.Digest {
	switch {
	case digester == nil:
		if data.IsZero() {
			return cryptkit.NewZeroSizeDigest("")
		}
		panic(throw.IllegalState())
	case data.IsZero():
		return digester.DigestBytes(nil)
	}
	hasher := digester.NewHasher()
	_, err := data.WriteTo(hasher)
	if err != nil {
		panic(throw.WithStackTop(err))
	}
	return hasher.SumToDigest()
}

// func (p *RawWithDigest) ProtoSize() int {
// 	p.digest.calcDigest(p._digestData, nil)
// 	return protokit.BinaryProtoSize(p.digestSize())
// }

func (p *RawWithDigest) SetPayload(data RawBinary, digester cryptkit.DataDigester) {
	p.digest.setDigester(digester, func() {
		p.data = data
	})
}

func (p *RawWithDigest) GetPayload() RawBinary {
	if p.digest.ready.WasStarted() {
		return p.data
	}
	return RawBinary{}
}

func (p *RawWithDigest) VerifyPayloadBytes(data []byte, digester cryptkit.DataDigester) error {
	b := RawBinary{}
	b.SetBytes(data)
	return p.VerifyPayload(b, digester)
}

func (p *RawWithDigest) VerifyPayload(data RawBinary, digester cryptkit.DataDigester) error {
	d0 := p.digest.GetDigest()

	var d1 cryptkit.Digest
	switch {
	case digester == nil:
		switch {
		case !data.IsEmpty():
			panic(throw.IllegalValue())
		case d0.IsEmpty():
			return nil
		}
	case !d0.IsEmpty():
		d1 := digestBy(data, digester)
		if m0 := d0.GetDigestMethod(); m0 != "" && m0 == d1.GetDigestMethod() {
			return throw.E("digest method mismatched", struct{ M0, M1 cryptkit.DigestMethod }{m0, d1.GetDigestMethod()})
		}
		if longbits.Equal(d0, d1) {
			return nil
		}
	case data.IsEmpty():
		return nil
	}
	return throw.E("digest mismatched", struct{ D0, D1 cryptkit.Digest }{d0, d1})
}

func (p *RawWithDigest) Equal(o *RawWithDigest) bool {
	switch {
	case p == o:
		return true
	case o == nil || p == nil:
		return false
	}
	return longbits.Equal(p.GetDigest(), o.GetDigest())
}
