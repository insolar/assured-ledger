package legacyadapter

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

const SHA3Digest224 = cryptkit.DigestMethod("sha3-224")

type Sha3Digester224 struct {
	scheme cryptography.PlatformCryptographyScheme
}

func NewSha3Digester224(scheme cryptography.PlatformCryptographyScheme) Sha3Digester224 {
	return Sha3Digester224{
		scheme: scheme,
	}
}

func (pd Sha3Digester224) DigestData(reader io.Reader) cryptkit.Digest {
	hasher := pd.scheme.ReferenceHasher()

	_, err := io.Copy(hasher, reader)
	if err != nil {
		panic(err)
	}

	bytes := hasher.Sum(nil)
	bits := longbits.NewBits224FromBytes(bytes)

	return cryptkit.NewDigest(bits, pd.GetDigestMethod())
}

func (pd Sha3Digester224) DigestBytes(bytes []byte) cryptkit.Digest {
	hasher := pd.scheme.ReferenceHasher()

	bytes = hasher.Hash(bytes)
	bits := longbits.NewBits224FromBytes(bytes)

	return cryptkit.NewDigest(bits, pd.GetDigestMethod())
}

func (pd Sha3Digester224) NewHasher() cryptkit.DigestHasher {
	return cryptkit.DigestHasher{BasicDigester: pd, Hash: pd.scheme.ReferenceHasher()}
}

func (pd Sha3Digester224) GetDigestSize() int {
	return 28
}

func (pd Sha3Digester224) GetDigestMethod() cryptkit.DigestMethod {
	return SHA3Digest224
}

