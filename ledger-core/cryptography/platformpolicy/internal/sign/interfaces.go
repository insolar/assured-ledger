package sign

import (
	"crypto"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
)

type AlgorithmProvider interface {
	DataSigner(crypto.PrivateKey, cryptography.Hasher) cryptography.Signer
	DigestSigner(crypto.PrivateKey) cryptography.Signer
	DataVerifier(crypto.PublicKey, cryptography.Hasher) cryptography.Verifier
	DigestVerifier(crypto.PublicKey) cryptography.Verifier
}
