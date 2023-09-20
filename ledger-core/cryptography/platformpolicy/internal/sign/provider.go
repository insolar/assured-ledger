package sign

import (
	"crypto"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
)

type ecdsaProvider struct {
}

func NewECDSAProvider() AlgorithmProvider {
	return &ecdsaProvider{}
}

func (p *ecdsaProvider) DataSigner(privateKey crypto.PrivateKey, hasher cryptography.Hasher) cryptography.Signer {
	return &ecdsaDataSignerWrapper{
		ecdsaDigestSignerWrapper: ecdsaDigestSignerWrapper{
			privateKey: MustConvertPrivateKeyToEcdsa(privateKey),
		},
		hasher: hasher,
	}
}
func (p *ecdsaProvider) DigestSigner(privateKey crypto.PrivateKey) cryptography.Signer {
	return &ecdsaDigestSignerWrapper{
		privateKey: MustConvertPrivateKeyToEcdsa(privateKey),
	}
}

func (p *ecdsaProvider) DataVerifier(publicKey crypto.PublicKey, hasher cryptography.Hasher) cryptography.Verifier {
	return &ecdsaDataVerifyWrapper{
		ecdsaDigestVerifyWrapper: ecdsaDigestVerifyWrapper{
			publicKey: MustConvertPublicKeyToEcdsa(publicKey),
		},
		hasher: hasher,
	}
}

func (p *ecdsaProvider) DigestVerifier(publicKey crypto.PublicKey) cryptography.Verifier {
	return &ecdsaDigestVerifyWrapper{
		publicKey: MustConvertPublicKeyToEcdsa(publicKey),
	}
}
