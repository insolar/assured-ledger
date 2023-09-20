package platformpolicy

import (
	"crypto"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy/internal/hash"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy/internal/sign"
)

type platformCryptographyScheme struct {
	hashProvider hash.AlgorithmProvider
	signProvider sign.AlgorithmProvider
}

func (pcs *platformCryptographyScheme) PublicKeySize() int {
	return sign.TwoBigIntBytesLength
}

func (pcs *platformCryptographyScheme) SignatureSize() int {
	return sign.TwoBigIntBytesLength
}

func (pcs *platformCryptographyScheme) ReferenceHashSize() int {
	return pcs.hashProvider.Hash224bits().Size()
}

func (pcs *platformCryptographyScheme) IntegrityHashSize() int {
	return pcs.hashProvider.Hash512bits().Size()
}

func (pcs *platformCryptographyScheme) ReferenceHasher() cryptography.Hasher {
	return pcs.hashProvider.Hash224bits()
}

func (pcs *platformCryptographyScheme) IntegrityHasher() cryptography.Hasher {
	return pcs.hashProvider.Hash512bits()
}

func (pcs *platformCryptographyScheme) DataSigner(privateKey crypto.PrivateKey, hasher cryptography.Hasher) cryptography.Signer {
	return pcs.signProvider.DataSigner(privateKey, hasher)
}

func (pcs *platformCryptographyScheme) DigestSigner(privateKey crypto.PrivateKey) cryptography.Signer {
	return pcs.signProvider.DigestSigner(privateKey)
}

func (pcs *platformCryptographyScheme) DataVerifier(publicKey crypto.PublicKey, hasher cryptography.Hasher) cryptography.Verifier {
	return pcs.signProvider.DataVerifier(publicKey, hasher)
}

func (pcs *platformCryptographyScheme) DigestVerifier(publicKey crypto.PublicKey) cryptography.Verifier {
	return pcs.signProvider.DigestVerifier(publicKey)
}

func NewPlatformCryptographyScheme() cryptography.PlatformCryptographyScheme {
	return &platformCryptographyScheme{
		hashProvider: hash.NewSHA3Provider(),
		signProvider: sign.NewECDSAProvider(),
	}
}
