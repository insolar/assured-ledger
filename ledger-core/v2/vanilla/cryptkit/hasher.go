// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cryptkit

import (
	"hash"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ hash.Hash = DigestHasher{}

type DigestHasher struct {
	BasicDigester
	hash.Hash
}

func (v DigestHasher) SumToDigest() Digest {
	return DigestOfHash(v.BasicDigester, v.Hash)
}

func DigestOfHash(digester BasicDigester, hasher hash.Hash) Digest {
	b := make([]byte, digester.GetDigestSize())
	if len(hasher.Sum(b)) != len(b) {
		panic(throw.IllegalValue())
	}
	return NewDigest(longbits.NewMutableFixedSize(b), digester.GetDigestMethod())
}
