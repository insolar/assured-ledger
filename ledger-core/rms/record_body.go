// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ GoGoSerializableWithText = &RecordBody{}


type RecordBody struct {
	rbd      RecordBodyDigest
	digester cryptkit.DataDigester
	payloads []RawBinary
}

func (p RecordBody) MarshalText() (text []byte, err error) {
	return nil, nil
}

func (p *RecordBody) Reset() {
	*p = RecordBody{}
}

func (p *RecordBody) SetDigester(digester cryptkit.DataDigester) {
	if p.digester != nil {
		p.rbd.Reset()
	}
	p.digester = digester
}

func (p *RecordBody) HasPayload() bool {
	return len(p.payloads) > 0
}

func (p *RecordBody) HasPayloadDigest() bool {
	return p.rbd.HasPayloadDigest()
}

func (p *RecordBody) SetPayload(body RawBinary) {
	if body.IsZero() {
		panic(throw.IllegalValue())
	}
	if len(p.payloads) == 0 {
		p.payloads = []RawBinary{body}
	} else {
		p.payloads[0] = body
	}
	p.rbd.Reset()
}

func (p *RecordBody) GetPayload() RawBinary {
	if len(p.payloads) == 0 {
		return RawBinary{}
	}
	return p.payloads[0]
}

func (p *RecordBody) GetExtensionPayloadCount() int {
	if n := len(p.payloads); n > 1 {
		return n - 1
	}
	return 0
}

func (p *RecordBody) GetExtensionDigestCount() int {
	return p.rbd.GetExtensionDigestCount()
}

func (p *RecordBody) AddExtensionPayload(body RawBinary) {
	switch n := len(p.payloads); {
	case n == 0:
		panic(throw.IllegalState())
	case body.IsZero():
		panic(throw.IllegalValue())
	case n == maxExtensionCount:
		panic(throw.IllegalState())
	}
	p.payloads = append(p.payloads, body)
	p.rbd.Reset()
}

func (p *RecordBody) GetExtensionPayload(index int) RawBinary {
	index++
	switch {
	case index < 1:
		panic(throw.IllegalValue())
	case index < len(p.payloads):
		return p.payloads[index]
	case index >= len(p.rbd.digests):
		panic(throw.IllegalValue())
	default:
		return RawBinary{}
	}
}

func (p *RecordBody) prepare() {
	nPayloads := len(p.payloads)
	switch {
	case nPayloads == 0:
		p.rbd.Reset()
		return
	case nPayloads == 1 && p.payloads[0].IsZero():
		m := cryptkit.DigestMethod("")
		if p.digester != nil {
			m = p.digester.GetDigestMethod()
		}
		p.rbd.digests = []cryptkit.Digest{cryptkit.NewZeroSizeDigest(m)}
		return
	case p.digester == nil:
		panic(throw.IllegalState())
	}

	for i := len(p.rbd.digests); i < nPayloads; i++ {
		hasher := p.digester.NewHasher()
		switch n, err := p.payloads[i].WriteTo(hasher); {
		case err != nil:
			panic(err)
		case n == 0 && i > 0:
			panic(throw.FailHere("extension can't be empty"))
		}
		digest := hasher.SumToDigest()
		p.rbd.digests = append(p.rbd.digests, digest)
	}
}

func (p *RecordBody) isPrepared() bool {
	switch {
	case len(p.payloads) != len(p.rbd.digests):
		return false
	case len(p.rbd.digests) > 0 && p.rbd.digests[0].IsZero():
		return false
	}
	return true
}

func (p *RecordBody) ensure() {
	if !p.isPrepared() {
		estSize := p._rawEstimateSize()
		p.prepare()
		if actSize := p._rawPreparedSize(); estSize != actSize {
			panic(throw.IllegalState())
		}
	}
}

func (p *RecordBody) ProtoSize() int {
	if p.isPrepared() {
		return protokit.BinaryProtoSize(p._rawPreparedSize())
	}
	return protokit.BinaryProtoSize(p._rawEstimateSize())
}

func (p *RecordBody) _rawEstimateSize() int {
	n := len(p.payloads)
	switch n {
	case 1:
		if p.payloads[0].IsZero() {
			return 0
		}
	case 0:
		return 0
	}
	if p.digester == nil {
		panic(throw.FailHere("digester not set"))
	}
	n *= p.digester.GetDigestSize()
	n++ // 1 byte for count
	return n
}

func (p *RecordBody) _rawPreparedSize() int {
	p.prepare()
	n := 0
	switch len(p.rbd.digests) {
	case 1:
		// body can be empty, hence first digest can be 0
		n = p.rbd.digests[0].FixedByteSize()
		if n == 0 {
			return 0
		}
	case 0:
		return 0
	default:
		// extensions can only exists with a body
		n = len(p.rbd.digests) * p.digester.GetDigestSize()
	}
	n++ // 1 byte for count
	return n
}

func (p *RecordBody) MarshalTo(b []byte) (int, error) {
	p.ensure()
	return protokit.BinaryMarshalTo(b, p.rbd._marshal)
}

func (p *RecordBody) MarshalToSizedBuffer(b []byte) (int, error) {
	p.ensure()
	return protokit.BinaryMarshalToSizedBuffer(b, func(b []byte) (int, error) {
		n := p._rawPreparedSize()
		return p.rbd._marshal(b[len(b)-n:])
	})
}

func (p *RecordBody) Unmarshal(b []byte) error {
	p.payloads = nil
	return p.rbd.Unmarshal(b)
}

func (p *RecordBody) VerifyAnyPayload(index int, data RawBinary) (err error) {
	if len(p.rbd.digests) == 0 && index == -1 {
		return p.verifyPayload(cryptkit.Digest{}, data)
	}
	return p.verifyPayload(p.rbd.digests[index+1], data)
}

func (p *RecordBody) PostUnmarshalVerifyAndAdd(data RawBinary) (err error) {
	n := len(p.payloads)
	switch {
	case len(p.rbd.digests) == 0:
		if n == 0 {
			err = p.verifyPayload(cryptkit.Digest{}, data)
			break
		}
		fallthrough
	case len(p.rbd.digests) <= n:
		return throw.FailHere("too many payloads")
	default:
		err = p.verifyPayload(p.rbd.digests[n], data)
	}

	if err != nil {
		p.payloads = append(p.payloads, RawBinary{})
		return err
	}
	p.payloads = append(p.payloads, data)
	return nil
}

func (p *RecordBody) IsPostUnmarshalCompleted() bool {
	n := len(p.rbd.digests)
	if n > 0 {
		return len(p.payloads) == n
	}
	return len(p.payloads) == 0 || p.payloads[0].IsZero()
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
		return p.rbd.Equal(nil)
	case p == nil:
		return o.rbd.Equal(nil)
	default:
		return p.rbd.Equal(&o.rbd)
	}
}

func (p *RecordBody) GetRecordPayloads() RecordPayloads {
	if len(p.payloads) != len(p.rbd.digests) {
		return RecordPayloads{
			payloads: p.payloads,
		}
	}

	return RecordPayloads{
		payloads: p.payloads,
		digester: p.digester,
		digests:  p.rbd.digests,
	}
}

func (p *RecordBody) CopyRecordPayloads(rp RecordPayloads) {
	switch {
	case len(p.payloads) > 0 || len(p.rbd.digests) > 0:
		panic(throw.IllegalState())
	case len(rp.payloads) != len(rp.digests):
		panic(throw.IllegalValue())
	}

	p.payloads = rp.payloads
	p.digester = rp.digester
	p.rbd.digests = rp.digests
}

func (p *RecordBody) SetRecordPayloads(rp RecordPayloads, digester cryptkit.DataDigester) error {
	switch {
	case p.digester == nil:
		p.digester = digester
	case digester == nil:
		digester = rp.digester
		fallthrough
	default:
		if digester.GetDigestMethod() != p.digester.GetDigestMethod() {
			panic(throw.IllegalState())
		}
	}

	n := len(rp.payloads)
	if n == 0 {
		p.payloads = nil
	} else {
		p.payloads = make([]RawBinary, 0, n)
		for i := range rp.payloads {
			if err := p.PostUnmarshalVerifyAndAdd(rp.payloads[i]); err != nil {
				return err
			}
		}
	}

	if !p.IsPostUnmarshalCompleted() {
		return throw.FailHere("payload number mismatched")
	}
	return nil
}

func (p *RecordBody) isEmptyForCopy() bool {
	switch {
	case p.digester != nil:
		return false
	case len(p.payloads) != 0:
		return false
	}
	return true
}
