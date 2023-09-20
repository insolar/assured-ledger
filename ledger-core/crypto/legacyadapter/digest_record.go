package legacyadapter

import (
	"hash"

	"github.com/insolar/assured-ledger/ledger-core/crypto"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

var _ crypto.RecordDigester = recordDualDigester{}
type recordDualDigester struct {
	Sha3Digester512
}

func (v recordDualDigester) NewDataAndRefHasher() cryptkit.DigestHasher {
	return cryptkit.DigestHasher{ BasicDigester: v.Sha3Digester512, Hash: dualHasher{
		v.scheme.IntegrityHasher(),
	}}
}

func (v recordDualDigester) GetDataAndRefDigests(hasher cryptkit.DigestHasher) (data, ref cryptkit.Digest) {
	dual := hasher.Hash.(dualHasher)
	hashBytes := cryptkit.ByteDigestOfHash(v.Sha3Digester512, dual.hashRec)

	dataHash := cryptkit.NewDigest(longbits.WrapBytes(hashBytes), SHA3Digest512)
	refHash := cryptkit.NewDigest(longbits.WrapBytes(hashBytes[:28]), SHA3Digest512as224)
	return dataHash, refHash
}

func (v recordDualDigester) GetRefDigestAndContinueData(hasher cryptkit.DigestHasher) (data cryptkit.DigestHasher, ref cryptkit.Digest) {
	dual := hasher.Hash.(dualHasher)
	hashBytes := cryptkit.ByteDigestOfHash(v.Sha3Digester512, dual.hashRec)
	refHash := cryptkit.NewDigest(longbits.WrapBytes(hashBytes[:28]), SHA3Digest512as224)

	return cryptkit.DigestHasher{ BasicDigester: v.Sha3Digester512, Hash: dual.hashRec},
		refHash
}


/*****************************/

var _ hash.Hash = dualHasher{}
type dualHasher struct {
	hashRec hash.Hash
}

func (v dualHasher) Write(b []byte) (n int, err error) {
	return v.hashRec.Write(b)
}

func (v dualHasher) Sum(b []byte) []byte {
	return v.hashRec.Sum(b)
}

func (v dualHasher) Reset() {
	v.hashRec.Reset()
}

func (v dualHasher) Size() int {
	return v.hashRec.Size()
}

func (v dualHasher) BlockSize() int {
	return v.hashRec.BlockSize()
}
