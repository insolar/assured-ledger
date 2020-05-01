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

type BlobBody struct {
	digest cryptkit.Digest
	data   *SerializableWithReference
}

func (p *BlobBody) ProtoSize() int {
	if p.digest.IsEmpty() && p.data != nil {
		p.digest = p.data.MustDigest()
		p.data = nil
	}
	n := p.digest.FixedByteSize()
	if n != 0 {
		n++ // protokit.BinaryMarker
	}
	return n
}

func (p *BlobBody) MarshalTo(b []byte) (int, error) {
	if p.digest.FixedByteSize() == 0 {
		return 0, nil
	}
	b[0] = protokit.BinaryMarker
	return p.digest.CopyTo(b), nil
}

func (p *BlobBody) MarshalToSizedBuffer(b []byte) (int, error) {
	n := p.digest.FixedByteSize()
	if n == 0 {
		return 0, nil
	}
	i := len(b) - n
	p.digest.CopyTo(b[i:])
	b[i-1] = protokit.BinaryMarker
	return n + 1, nil
}

func (p *BlobBody) Unmarshal(b []byte) error {
	if len(b) == 0 {
		p.digest = cryptkit.NewDigest(longbits.EmptyByteString, "")
	}
	if b[0] != protokit.BinaryMarker {
		return throw.IllegalValue()
	}
	p.digest = cryptkit.NewDigest(longbits.NewImmutableFixedSize(b[1:]), "")
	return nil
}
