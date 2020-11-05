// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rmsbox

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const maxExtensionCount = 0x7F

type RecordBodyDigests struct {
	digests  []cryptkit.Digest
}

func (p RecordBodyDigests) MarshalText() (text []byte, err error) {
	return nil, nil
}

func (p *RecordBodyDigests) Reset() {
	*p = RecordBodyDigests{}
}

func (p *RecordBodyDigests) IsEmpty() bool {
	return len(p.digests) == 0
}

func (p *RecordBodyDigests) GetDigest(index int) cryptkit.Digest {
	if n := len(p.digests); n <= index || index < 0 {
		return cryptkit.Digest{}
	}
	return p.digests[index]
}

func (p *RecordBodyDigests) AddDigest(d0 cryptkit.Digest) {
	switch n := len(p.digests); {
	case n >= maxExtensionCount:
		panic(throw.IllegalState())
	case n != 0	&& d0.IsEmpty():
		panic(throw.IllegalValue())
	}
	p.digests = append(p.digests, d0)
}

func (p *RecordBodyDigests) Count() int {
	return len(p.digests)
}

func (p *RecordBodyDigests) ProtoSize() int {
	return protokit.BinaryProtoSize(p.rawProtoSize())
}

func (p *RecordBodyDigests) rawProtoSize() int {
	if n := len(p.digests); n > 0 {
		return 1 + n * p.digests[0].FixedByteSize()
	}
	return 0
}

func (p *RecordBodyDigests) MarshalTo(b []byte) (int, error) {
	return protokit.BinaryMarshalTo(b, false, p._marshal)
}

func (p *RecordBodyDigests) _marshal(b []byte) (int, error) {
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

func (p *RecordBodyDigests) MarshalToSizedBuffer(b []byte) (int, error) {
	return protokit.BinaryMarshalToSizedBuffer(b, false, func(b []byte) (int, error) {
		n := p.rawProtoSize()
		return p._marshal(b[len(b)-n:])
	})
}

func (p *RecordBodyDigests) Unmarshal(b []byte) error {
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
			digest := cryptkit.NewDigest(longbits.CopyBytes(b[n:n+digestSize]), "")
			n += digestSize
			p.digests = append(p.digests, digest)
		}

		return nil
	})
}

func (p *RecordBodyDigests) Equal(o *RecordBodyDigests) bool {
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

func (p *RecordBodyDigests) VerifyDigest(index int, data RawBinary, digester cryptkit.DataDigester) error {
	return verifyBodyDigest(p.GetDigest(index), data, digester)
}

func verifyBodyDigest(d0 cryptkit.Digest, data RawBinary, digester cryptkit.DataDigester) error {
	switch {
	case digester == nil:
		switch {
		case !data.IsEmpty():
			panic(throw.IllegalValue())
		case d0.IsEmpty():
			return nil
		}
	case !d0.IsEmpty():
		hasher := digester.NewHasher()
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
