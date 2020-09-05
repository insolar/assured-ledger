// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package legacyadapter

import (
	"crypto"
	"crypto/ecdsa"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

const (
	SECP256r1Sign = cryptkit.SigningMethod("secp256r1")
)

func NewECDSAPublicKeyStore(skh cryptkit.SigningKeyHolder) *ECDSAPublicKeyStore {
	return &ECDSAPublicKeyStore{
		publicKey: skh.(*ECDSASignatureKeyHolder).publicKey,
	}
}

func NewECDSAPublicKeyStoreFromPK(pk crypto.PublicKey) *ECDSAPublicKeyStore {
	return &ECDSAPublicKeyStore{
		publicKey: pk.(*ecdsa.PublicKey),
	}
}

type ECDSAPublicKeyStore struct {
	publicKey *ecdsa.PublicKey
}

func (p *ECDSAPublicKeyStore) CryptoPublicKey() crypto.PublicKey {
	return p.publicKey
}

func (*ECDSAPublicKeyStore) PublicKeyStore() {}


func NewECDSASecretKeyStore(privateKey *ecdsa.PrivateKey) *ECDSASecretKeyStore {
	return &ECDSASecretKeyStore{
		privateKey: privateKey,
	}
}

type ECDSASecretKeyStore struct {
	privateKey *ecdsa.PrivateKey
}

func (ks *ECDSASecretKeyStore) PrivateKeyStore() {}

func (ks *ECDSASecretKeyStore) AsPublicKeyStore() cryptkit.PublicKeyStore {
	return &ECDSAPublicKeyStore{&ks.privateKey.PublicKey }
}

type ECDSADigestSigner struct {
	scheme     cryptography.PlatformCryptographyScheme
	privateKey *ecdsa.PrivateKey
}

func NewECDSADigestSigner(sks cryptkit.SecretKeyStore, scheme cryptography.PlatformCryptographyScheme) *ECDSADigestSigner {
	return &ECDSADigestSigner{
		scheme:     scheme,
		privateKey: sks.(*ECDSASecretKeyStore).privateKey,
	}
}

func NewECDSADigestSignerFromSK(sk crypto.PrivateKey, scheme cryptography.PlatformCryptographyScheme) *ECDSADigestSigner {
	return &ECDSADigestSigner{
		scheme:     scheme,
		privateKey: sk.(*ecdsa.PrivateKey),
	}
}

func (ds *ECDSADigestSigner) SignDigest(digest cryptkit.Digest) cryptkit.Signature {
	digestBytes := longbits.AsBytes(digest)

	signer := ds.scheme.DigestSigner(ds.privateKey)

	sig, err := signer.Sign(digestBytes)
	if err != nil {
		panic("Failed to create signature")
	}

	sigBytes := sig.Bytes()
	bits := longbits.NewBits512FromBytes(sigBytes)

	return cryptkit.NewSignature(bits, digest.GetDigestMethod().SignedBy(ds.GetSigningMethod()))
}

func (ds *ECDSADigestSigner) GetSigningMethod() cryptkit.SigningMethod {
	return SECP256r1Sign
}

type ECDSASignatureVerifier struct {
	scheme    cryptography.PlatformCryptographyScheme
	publicKey *ecdsa.PublicKey
}

func NewECDSASignatureVerifier(pcs cryptography.PlatformCryptographyScheme, pks cryptkit.PublicKeyStore) *ECDSASignatureVerifier {
	return &ECDSASignatureVerifier{
		scheme:    pcs,
		publicKey: pks.(*ECDSAPublicKeyStore).publicKey,
	}
}

func (sv *ECDSASignatureVerifier) GetDefaultSigningMethod() cryptkit.SigningMethod {
	return SECP256r1Sign
}

func (sv *ECDSASignatureVerifier) IsDigestMethodSupported(method cryptkit.DigestMethod) bool {
	return method == SHA3Digest512
}

func (sv *ECDSASignatureVerifier) IsSigningMethodSupported(method cryptkit.SigningMethod) bool {
	return method == SECP256r1Sign
}

func (sv *ECDSASignatureVerifier) IsValidDigestSignature(digest cryptkit.DigestHolder, signature cryptkit.SignatureHolder) bool {
	method := signature.GetSignatureMethod()
	if digest.GetDigestMethod() != method.DigestMethod() || !sv.IsSigningMethodSupported(method.SigningMethod()) {
		return false
	}

	digestBytes := longbits.AsBytes(digest)
	signatureBytes := longbits.AsBytes(signature)

	verifier := sv.scheme.DigestVerifier(sv.publicKey)
	return verifier.Verify(cryptography.SignatureFromBytes(signatureBytes), digestBytes)
}

func (sv *ECDSASignatureVerifier) IsValidDataSignature(data io.Reader, signature cryptkit.SignatureHolder) bool {
	digester := NewSha3Digester512(sv.scheme)
	if digester.GetDigestMethod() != signature.GetSignatureMethod().DigestMethod() {
		return false
	}
	digest := digester.DigestData(data)
	return sv.IsValidDigestSignature(digest, signature)
}

type ECDSASignatureKeyHolder struct {
	longbits.Bits512
	publicKey *ecdsa.PublicKey
}

func NewECDSASignatureKeyHolder(publicKey *ecdsa.PublicKey, processor cryptography.KeyProcessor) *ECDSASignatureKeyHolder {
	publicKeyBytes, err := processor.ExportPublicKeyBinary(publicKey)
	if err != nil {
		panic(err)
	}

	bits := longbits.NewBits512FromBytes(publicKeyBytes)
	return &ECDSASignatureKeyHolder{
		Bits512:   bits,
		publicKey: publicKey,
	}
}

func NewECDSASignatureKeyHolderFromBits(publicKeyBytes longbits.Bits512, processor cryptography.KeyProcessor) *ECDSASignatureKeyHolder {
	publicKey, err := processor.ImportPublicKeyBinary(publicKeyBytes.AsBytes())
	if err != nil {
		panic(err)
	}

	return &ECDSASignatureKeyHolder{
		Bits512:   publicKeyBytes,
		publicKey: publicKey.(*ecdsa.PublicKey),
	}
}

func (kh *ECDSASignatureKeyHolder) GetSigningMethod() cryptkit.SigningMethod {
	return SECP256r1Sign
}

func (kh *ECDSASignatureKeyHolder) GetSigningKeyType() cryptkit.SigningKeyType {
	return cryptkit.PublicAsymmetricKey
}

func (kh *ECDSASignatureKeyHolder) Equals(other cryptkit.SigningKeyHolder) bool {
	okh, ok := other.(*ECDSASignatureKeyHolder)
	if !ok {
		return false
	}

	return kh.Bits512 == okh.Bits512
}
