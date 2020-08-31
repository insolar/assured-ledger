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

const maxExtensionCount = 0x7F

type RecordBodyDigest struct {
	digests  []cryptkit.Digest
}

func (p RecordBodyDigest) MarshalText() (text []byte, err error) {
	return nil, nil
}

func (p *RecordBodyDigest) Reset() {
	*p = RecordBodyDigest{}
}

func (p *RecordBodyDigest) isEmpty() bool {
	return len(p.digests) == 0
}

func (p *RecordBodyDigest) HasPayloadDigest() bool {
	return len(p.digests) > 0
}

func (p *RecordBodyDigest) GetExtensionDigestCount() int {
	if n := len(p.digests); n > 1 {
		return n - 1
	}
	return 0
}

func (p *RecordBodyDigest) ProtoSize() int {
	return protokit.BinaryProtoSize(p.rawProtoSize())
}

func (p *RecordBodyDigest) rawProtoSize() int {
	if n := len(p.digests); n > 0 {
		return 1 + n * p.digests[0].FixedByteSize()
	}
	return 0
}

func (p *RecordBodyDigest) MarshalTo(b []byte) (int, error) {
	return protokit.BinaryMarshalTo(b, p._marshal)
}

func (p *RecordBodyDigest) _marshal(b []byte) (int, error) {
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

func (p *RecordBodyDigest) MarshalToSizedBuffer(b []byte) (int, error) {
	return protokit.BinaryMarshalToSizedBuffer(b, func(b []byte) (int, error) {
		n := p.rawProtoSize()
		return p._marshal(b[len(b)-n:])
	})
}

func (p *RecordBodyDigest) Unmarshal(b []byte) error {
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

func (p *RecordBodyDigest) GetRecordPayloads() RecordPayloads {
	return RecordPayloads{}
}

func (p *RecordBodyDigest) SetRecordPayloads(rp RecordPayloads, _ cryptkit.DataDigester) error {
	if len(rp.payloads) != 0 {
		return throw.FailHere("payload(s) not allowed")
	}
	return nil
}

func (p *RecordBodyDigest) Equal(o *RecordBodyDigest) bool {
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
