// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ GoGoSerializable = &RecordBody{}

const maxExtensionCount = 0x7F

type RecordBody struct {
	digester cryptkit.DataDigester
	payloads []RawBinary
	digests  []cryptkit.Digest
}

func (RecordBody) RecordBody() {}

func (p *RecordBody) Reset() {
	*p = RecordBody{}
}

func (p *RecordBody) SetDigester(digester cryptkit.DataDigester) {
	if p.digester != nil {
		p.digests = nil
	}
	p.digester = digester
}

func (p *RecordBody) SetPayload(body RawBinary) {
	if len(p.payloads) == 0 {
		p.payloads = []RawBinary{body}
	} else {
		p.payloads[0] = body
	}
	p.digests = nil
}

func (p *RecordBody) AddExtension(body RawBinary) {
	switch n := len(p.payloads); {
	case n == 0:
		panic(throw.IllegalState())
	case body.IsZero():
		panic(throw.IllegalValue())
	case n == maxExtensionCount:
		panic(throw.IllegalState())
	}
	p.payloads = append(p.payloads, body)
	p.digests = nil
}

func (p *RecordBody) prepare() {
	nPayloads := len(p.payloads)
	switch {
	case nPayloads == 0:
		p.digests = nil
		return
	case nPayloads == 1 && p.payloads[0].IsZero():
		m := cryptkit.DigestMethod("")
		if p.digester != nil {
			m = p.digester.GetDigestMethod()
		}
		p.digests = []cryptkit.Digest{cryptkit.NewZeroSizeDigest(m)}
		return
	case p.digester == nil:
		panic(throw.IllegalState())
	}

	for i := len(p.digests); i < nPayloads; i++ {
		hasher := p.digester.NewHasher()
		switch n, err := p.payloads[i].WriteTo(hasher); {
		case err != nil:
			panic(err)
		case n == 0 && i > 0:
			panic(throw.FailHere("extension can't be empty"))
		}
		digest := hasher.SumToDigest()
		p.digests = append(p.digests, digest)
	}
}

func (p *RecordBody) ensure() {
	switch {
	case len(p.payloads) != len(p.digests):
		panic(throw.IllegalState())
	case len(p.digests) > 0 && p.digests[0].IsZero():
		panic(throw.IllegalState())
	}
}

func (p *RecordBody) ProtoSize() int {
	return protokit.BinaryProtoSize(p._rawSize())
}

func (p *RecordBody) _rawSize() int {
	p.prepare()
	n := 0
	switch len(p.digests) {
	case 1:
		// body can be empty, hence first digest can be 0
		n = p.digests[0].FixedByteSize()
	case 0:
		n = 0
	default:
		// extensions can only exists with a body
		n = len(p.digests) * p.digester.GetDigestSize()
	}
	if n > 0 {
		n++ // 1 byte for count
	}
	return n
}

func (p *RecordBody) MarshalTo(b []byte) (int, error) {
	p.ensure()
	return protokit.BinaryMarshalTo(b, p._marshal)
}

func (p *RecordBody) _marshal(b []byte) (int, error) {
	n := len(p.digests)
	switch {
	case n == 0:
		return 0, nil
	case n > maxExtensionCount:
		panic(throw.IllegalState())
	}
	b[0] = uint8(n)
	n = 1
	for i := range p.digests {
		n += p.digests[i].CopyTo(b[n:])
	}
	return n, nil
}

func (p *RecordBody) MarshalToSizedBuffer(b []byte) (int, error) {
	p.ensure()
	return protokit.BinaryMarshalToSizedBuffer(b, func(b []byte) (int, error) {
		n := p._rawSize()
		return p._marshal(b[len(b)-n:])
	})
}

func (p *RecordBody) Unmarshal(b []byte) error {
	return protokit.BinaryUnmarshal(b, func(b []byte) error {
		n := len(b)
		if n == 0 {
			p.digests = nil
			return nil
		}
		n--
		count := int(b[0])
		switch {
		case count == 0 || count > maxExtensionCount:
			return throw.FailHere("wrong count")
		case n == 0:
			if count != 1 {
				return throw.FailHere("empty extensions")
			}
			p.digests = []cryptkit.Digest{cryptkit.NewZeroSizeDigest("")}
			return nil
		case n%count != 0:
			return throw.FailHere("count and size mismatched")
		}
		digestSize := n / count

		n = 1
		for ; count > 0; count-- {
			digest := cryptkit.NewDigest(longbits.NewImmutableFixedSize(b[n:n+digestSize]), "")
			n += digestSize
			p.digests = append(p.digests, digest)
		}

		return nil
	})
}

func (p *RecordBody) VerifyPayloadBytes(data []byte) error {
	b := RawBinary{}
	b.SetBytes(data)
	return p.VerifyPayload(b)
}

func (p *RecordBody) VerifyPayload(data RawBinary) error {
	if len(p.digests) == 0 {
		return p.verifyPayload(cryptkit.Digest{}, data)
	}
	return p.verifyPayload(p.digests[0], data)
}

func (p *RecordBody) verifyPayload(d0 cryptkit.Digest, data RawBinary) error {
	switch {
	case p.digester == nil:
		switch {
		case !data.IsEmpty():
			panic(throw.IllegalValue())
		case d0.IsEmpty():
			return nil
		}
	case !d0.IsEmpty():
		hasher := p.digester.NewHasher()
		_, _ = data.WriteTo(hasher)
		d1 := hasher.SumToDigest()

		if m0 := d0.GetDigestMethod(); m0 != "" && m0 == d1.GetDigestMethod() {
			return throw.E("digest method mismatched", struct{ M0, M1 cryptkit.DigestMethod }{m0, d1.GetDigestMethod()})
		}
		if longbits.Equal(d0, d1) {
			return nil
		}
		return throw.E("digest mismatched", struct{ D0, D1 cryptkit.Digest }{d0, d1})
	case data.IsEmpty():
		return nil
	}
	return throw.E("digest unmatched", struct{ D0 cryptkit.Digest }{d0})
}

func (p *RecordBody) Equal(o *RecordBody) bool {
	switch {
	case p == o:
		return true
	case o == nil:
		return len(p.digests) == 0
	case p == nil:
		return len(o.digests) == 0
	case len(p.digests) != len(o.digests):
		return false
	}
	for i := range p.digests {
		if !longbits.Equal(p.digests[i], o.digests[i]) {
			return false
		}
	}
	return true
}
