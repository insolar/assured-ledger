package sign

import (
	"crypto/ecdsa"
	"crypto/rand"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/log/global"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type ecdsaDigestSignerWrapper struct {
	privateKey *ecdsa.PrivateKey
}

func (sw *ecdsaDigestSignerWrapper) Sign(digest []byte) (*cryptography.Signature, error) {
	r, s, err := ecdsa.Sign(rand.Reader, sw.privateKey, digest)
	if err != nil {
		return nil, errors.W(err, "[ DataSigner ] could't sign data")
	}

	ecdsaSignature := SerializeTwoBigInt(r, s)

	signature := cryptography.SignatureFromBytes(ecdsaSignature)
	return &signature, nil
}

type ecdsaDataSignerWrapper struct {
	ecdsaDigestSignerWrapper
	hasher cryptography.Hasher
}

func (sw *ecdsaDataSignerWrapper) Sign(data []byte) (*cryptography.Signature, error) {
	return sw.ecdsaDigestSignerWrapper.Sign(sw.hasher.Hash(data))
}

type ecdsaDigestVerifyWrapper struct {
	publicKey *ecdsa.PublicKey
}

func (sw *ecdsaDigestVerifyWrapper) Verify(signature cryptography.Signature, data []byte) bool {
	if signature.Bytes() == nil {
		return false
	}
	r, s, err := DeserializeTwoBigInt(signature.Bytes())
	if err != nil {
		global.Error(err)
		return false
	}

	return ecdsa.Verify(sw.publicKey, data, r, s)
}

type ecdsaDataVerifyWrapper struct {
	ecdsaDigestVerifyWrapper
	hasher cryptography.Hasher
}

func (sw *ecdsaDataVerifyWrapper) Verify(signature cryptography.Signature, data []byte) bool {
	return sw.ecdsaDigestVerifyWrapper.Verify(signature, sw.hasher.Hash(data))
}
