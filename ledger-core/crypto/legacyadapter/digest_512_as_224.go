package legacyadapter

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

const SHA3Digest512as224 = cryptkit.DigestMethod("sha3-512-224")

type Sha3Digester512as224 struct {
	scheme cryptography.PlatformCryptographyScheme
}

func NewSha3Digester512as224(scheme cryptography.PlatformCryptographyScheme) Sha3Digester512as224 {
	return Sha3Digester512as224{
		scheme: scheme,
	}
}

func (pd Sha3Digester512as224) DigestData(reader io.Reader) cryptkit.Digest {
	hasher := pd.scheme.IntegrityHasher()

	_, err := io.Copy(hasher, reader)
	if err != nil {
		panic(err)
	}

	bytes := hasher.Sum(nil)
	bits := longbits.NewBits224FromBytes(bytes[:28])

	return cryptkit.NewDigest(bits, pd.GetDigestMethod())
}

func (pd Sha3Digester512as224) DigestBytes(bytes []byte) cryptkit.Digest {
	hasher := pd.scheme.IntegrityHasher()

	bytes = hasher.Hash(bytes)
	bits := longbits.NewBits224FromBytes(bytes[:28])

	return cryptkit.NewDigest(bits, pd.GetDigestMethod())
}

func (pd Sha3Digester512as224) NewHasher() cryptkit.DigestHasher {
	return cryptkit.DigestHasher{BasicDigester: pd, Hash: pd.scheme.IntegrityHasher()}
}

func (pd Sha3Digester512as224) GetDigestSize() int {
	return 28
}

func (pd Sha3Digester512as224) GetDigestMethod() cryptkit.DigestMethod {
	return SHA3Digest512as224
}

