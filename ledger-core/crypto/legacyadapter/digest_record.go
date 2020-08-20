// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package legacyadapter

import (
	"hash"

	"github.com/insolar/assured-ledger/ledger-core/crypto"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

var _ crypto.RecordDigester = recordDualDigester{}
type recordDualDigester struct {
	Sha3Digester512
}

func (v recordDualDigester) NewDataAndRefHasher() cryptkit.DigestHasher {
	return cryptkit.DigestHasher{ BasicDigester: v.Sha3Digester512, Hash: dualHasher{
		v.scheme.IntegrityHasher(),
		v.scheme.ReferenceHasher(),
	}}
}

func (v recordDualDigester) GetDataAndRefDigests(hasher cryptkit.DigestHasher) (data, ref cryptkit.Digest) {
	dual := hasher.Hash.(dualHasher)
	return cryptkit.DigestOfHash(v.Sha3Digester512, dual.hashRec), cryptkit.DigestOfHash(Sha3Digester224{}, dual.hashRef)
}

func (v recordDualDigester) GetRefDigestAndContinueData(hasher cryptkit.DigestHasher) (data cryptkit.DigestHasher, ref cryptkit.Digest) {
	dual := hasher.Hash.(dualHasher)
	return cryptkit.DigestHasher{ BasicDigester: v.Sha3Digester512, Hash: dual.hashRec},
		cryptkit.DigestOfHash(Sha3Digester224{}, dual.hashRef)
}


/*****************************/

var _ hash.Hash = dualHasher{}
type dualHasher struct {
	hashRec, hashRef hash.Hash
}

func (v dualHasher) Write(b []byte) (n int, err error) {
	if n, err = v.hashRec.Write(b); err != nil {
		return
	}
	return v.hashRef.Write(b)
}

func (v dualHasher) Sum(b []byte) []byte {
	return v.hashRec.Sum(b)
}

func (v dualHasher) Reset() {
	v.hashRec.Reset()
	v.hashRef.Reset()
}

func (v dualHasher) Size() int {
	return v.hashRec.Size()
}

func (v dualHasher) BlockSize() int {
	return v.hashRec.BlockSize()
}
