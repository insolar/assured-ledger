// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cryptkit

import (
	"hash"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ hash.Hash = DigestHasher{}

type DigestHasher struct {
	BasicDigester
	hash.Hash
}

func (v DigestHasher) DigestReader(r io.Reader) DigestHasher {
	if _, err := io.Copy(v.Hash, r); err != nil {
		panic(err)
	}
	return v
}

func (v DigestHasher) DigestOf(w io.WriterTo) DigestHasher {
	if _, err := w.WriteTo(v.Hash); err != nil {
		panic(err)
	}
	return v
}

func (v DigestHasher) DigestBytes(b []byte) DigestHasher {
	if _, err := v.Hash.Write(b); err != nil {
		panic(err)
	}
	return v
}

func (v DigestHasher) SumToDigest() Digest {
	return DigestOfHash(v.BasicDigester, v.Hash)
}

func DigestOfHash(digester BasicDigester, hasher hash.Hash) Digest {
	n := digester.GetDigestSize()
	if h := hasher.Sum(make([]byte, 0, n)); len(h) != n {
		panic(throw.IllegalValue())
	} else {
		return NewDigest(longbits.NewMutableFixedSize(h), digester.GetDigestMethod())
	}
}
