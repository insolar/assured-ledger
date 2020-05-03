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

var _ GoGoSerializable = &BodyWithDigest{}
var _ DigestDispenser = &BodyWithDigest{}

type BodyWithDigest struct {
	data   Serializable
	digest digestDispenser
}

func (p *BodyWithDigest) GetDigest() cryptkit.Digest {
	return p.digest.GetDigest()
}

func (p *BodyWithDigest) MustDigest() cryptkit.Digest {
	return p.digest.MustDigest()
}

// func (p *BodyWithDigest) SetDigester(digester cryptkit.DataDigester) {
// 	p.digest.setDigester(digester)
// }

func (p *BodyWithDigest) ProtoSize() int {
	p.digest.calcDigest(p._digestData)
	return p._protoSize()
}

func (p *BodyWithDigest) _digestData(digester cryptkit.DataDigester) cryptkit.Digest {
	if p.data == nil {
		return digester.DigestBytes(nil)
	}
	data, err := p.data.Marshal()
	if err != nil {
		panic(throw.WithStackTop(err))
	}
	return digester.DigestBytes(data)
}

func (p *BodyWithDigest) _protoSize() int {
	n := p.digest.digest.FixedByteSize()
	if n != 0 {
		n++ // protokit.BinaryMarker
	}
	return n
}

func (p *BodyWithDigest) mustProtoSize() int {
	if !p.digest.isReady() {
		panic(throw.IllegalState())
	}
	return p._protoSize()
}

func (p *BodyWithDigest) MarshalTo(b []byte) (int, error) {
	n := p.mustProtoSize()
	if n == 0 {
		return 0, nil
	}
	b[0] = protokit.BinaryMarker
	return p.digest.digest.CopyTo(b), nil
}

func (p *BodyWithDigest) MarshalToSizedBuffer(b []byte) (int, error) {
	n := p.mustProtoSize()
	if n == 0 {
		return 0, nil
	}
	i := len(b) - n
	p.digest.digest.CopyTo(b[i:])
	b[i-1] = protokit.BinaryMarker
	return n + 1, nil
}

func (p *BodyWithDigest) Unmarshal(b []byte) error {
	if len(b) == 0 {
		p.digest.setDigest(cryptkit.NewDigest(longbits.EmptyByteString, ""))
		return nil
	}
	if b[0] != protokit.BinaryMarker {
		return throw.IllegalValue()
	}
	p.digest.setDigest(cryptkit.NewDigest(longbits.NewImmutableFixedSize(b[1:]), ""))
	return nil
}
