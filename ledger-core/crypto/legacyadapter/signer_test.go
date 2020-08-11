// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package legacyadapter

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

var (
	processor  = platformpolicy.NewKeyProcessor()
	scheme     = platformpolicy.NewPlatformCryptographyScheme()
	pk, _      = processor.GeneratePrivateKey()
	privateKey = pk.(*ecdsa.PrivateKey)
	publicKey  = &privateKey.PublicKey
)

func TestNewECDSAPublicKeyStore(t *testing.T) {
	ks := NewECDSAPublicKeyStoreFromPK(publicKey)

	require.Implements(t, (*cryptkit.PublicKeyStore)(nil), ks)

	require.Equal(t, ks.publicKey, publicKey)
}

func TestECDSAPublicKeyStore_PublicKeyStore(t *testing.T) {
	ks := NewECDSAPublicKeyStoreFromPK(publicKey)

	ks.PublicKeyStore()
}

func TestNewECDSASecretKeyStore(t *testing.T) {
	ks := NewECDSASecretKeyStore(privateKey)

	require.Implements(t, (*cryptkit.SecretKeyStore)(nil), ks)

	require.Equal(t, ks.privateKey, privateKey)
}

func TestECDSASecretKeyStore_PrivateKeyStore(t *testing.T) {
	ks := NewECDSASecretKeyStore(privateKey)

	ks.PrivateKeyStore()
}

func TestECDSASecretKeyStore_AsPublicKeyStore(t *testing.T) {
	ks := NewECDSASecretKeyStore(privateKey)

	expected := NewECDSAPublicKeyStoreFromPK(publicKey)

	require.Equal(t, expected, ks.AsPublicKeyStore())
}

func TestNewECDSADigestSigner(t *testing.T) {
	ds := NewECDSADigestSigner(NewECDSASecretKeyStore(privateKey), scheme)

	require.Implements(t, (*cryptkit.DigestSigner)(nil), ds)

	require.Equal(t, ds.privateKey, privateKey)
	require.Equal(t, ds.scheme, scheme)
}

func TestECDSADigestSigner_SignDigest(t *testing.T) {
	ds := NewECDSADigestSigner(NewECDSASecretKeyStore(privateKey), scheme)
	digester := NewSha3Digester512(scheme)

	verifier := scheme.DigestVerifier(publicKey)

	b := make([]byte, 120)
	_, _ = rand.Read(b)
	reader := bytes.NewReader(b)

	digest := digester.DigestData(reader)
	digestBytes := longbits.AsBytes(digest)

	signature := ds.SignDigest(digest)
	require.Equal(t, scheme.SignatureSize(), signature.FixedByteSize())
	require.Equal(t, signature.GetSignatureMethod(), SHA3Digest512.SignedBy(SECP256r1Sign))

	signatureBytes := longbits.AsBytes(signature)

	require.True(t, verifier.Verify(cryptography.SignatureFromBytes(signatureBytes), digestBytes))
}

func TestECDSADigestSigner_GetSignMethod(t *testing.T) {
	ds := NewECDSADigestSigner(NewECDSASecretKeyStore(privateKey), scheme)

	require.Equal(t, ds.GetSigningMethod(), SECP256r1Sign)
}

func TestNewECDSASignatureVerifier(t *testing.T) {
	dv := NewECDSASignatureVerifier(scheme, NewECDSAPublicKeyStoreFromPK(publicKey))

	require.Implements(t, (*cryptkit.SignatureVerifier)(nil), dv)

	require.Equal(t, dv.scheme, scheme)
	require.Equal(t, dv.publicKey, publicKey)
}

func TestECDSASignatureVerifier_IsDigestMethodSupported(t *testing.T) {
	dv := NewECDSASignatureVerifier(scheme, NewECDSAPublicKeyStoreFromPK(publicKey))

	require.True(t, dv.IsDigestMethodSupported(SHA3Digest512))
	require.False(t, dv.IsDigestMethodSupported("SOME DIGEST METHOD"))
}

func TestECDSASignatureVerifier_IsSignMethodSupported(t *testing.T) {
	dv := NewECDSASignatureVerifier(scheme, NewECDSAPublicKeyStoreFromPK(publicKey))

	require.True(t, dv.IsSignMethodSupported(SECP256r1Sign))
	require.False(t, dv.IsSignMethodSupported("SOME SIGN METHOD"))
}

func TestECDSASignatureVerifier_IsSignOfSignatureMethodSupported(t *testing.T) {
	dv := NewECDSASignatureVerifier(scheme, NewECDSAPublicKeyStoreFromPK(publicKey))

	require.True(t, dv.IsSignOfSignatureMethodSupported(SHA3Digest512.SignedBy(SECP256r1Sign)))
	require.False(t, dv.IsSignOfSignatureMethodSupported("SOME SIGNATURE METHOD"))
	require.False(t, dv.IsSignOfSignatureMethodSupported(SHA3Digest512.SignedBy("SOME SIGN METHOD")))
	require.True(t, dv.IsSignOfSignatureMethodSupported(cryptkit.DigestMethod("SOME DIGEST METHOD").SignedBy(SECP256r1Sign)))
}

func TestECDSASignatureVerifier_IsValidDigestSignature(t *testing.T) {
	dv := NewECDSASignatureVerifier(scheme, NewECDSAPublicKeyStoreFromPK(publicKey))

	signer := scheme.DigestSigner(privateKey)

	b := make([]byte, 120)
	_, _ = rand.Read(b)
	reader := bytes.NewReader(b)

	digester := NewSha3Digester512(scheme)
	digest := digester.DigestData(reader)
	digestBytes := longbits.AsBytes(digest)

	signature, _ := signer.Sign(digestBytes)

	sig := cryptkit.NewSignature(longbits.NewBits512FromBytes(signature.Bytes()), SHA3Digest512.SignedBy(SECP256r1Sign))

	require.True(t, dv.IsValidDigestSignature(digest, sig))
}

func TestECDSASignatureVerifier_IsValidDigestSignature_InvalidMethod(t *testing.T) {
	dv := NewECDSASignatureVerifier(scheme, NewECDSAPublicKeyStoreFromPK(publicKey))

	signer := scheme.DigestSigner(privateKey)

	b := make([]byte, 120)
	_, _ = rand.Read(b)
	reader := bytes.NewReader(b)

	digester := NewSha3Digester512(scheme)
	digest := digester.DigestData(reader)
	digestBytes := longbits.AsBytes(digest)

	signature, _ := signer.Sign(digestBytes)
	bits := longbits.NewBits512FromBytes(signature.Bytes())

	sig1 := cryptkit.NewSignature(bits, SHA3Digest512.SignedBy(SECP256r1Sign))
	require.True(t, dv.IsValidDigestSignature(digest, sig1))

	sig2 := cryptkit.NewSignature(bits, "SOME DIGEST METHOD")
	require.False(t, dv.IsValidDigestSignature(digest, sig2))

	sig3 := cryptkit.NewSignature(bits, SHA3Digest512.SignedBy("SOME SIGN METHOD"))
	require.False(t, dv.IsValidDigestSignature(digest, sig3))

	sig4 := cryptkit.NewSignature(bits, cryptkit.DigestMethod("SOME DIGEST METHOD").SignedBy(SECP256r1Sign))
	require.False(t, dv.IsValidDigestSignature(digest, sig4))
}

func TestECDSASignatureVerifier_IsValidDataSignature(t *testing.T) {
	dv := NewECDSASignatureVerifier(scheme, NewECDSAPublicKeyStoreFromPK(publicKey))

	signer := scheme.DigestSigner(privateKey)

	b := make([]byte, 120)
	_, _ = rand.Read(b)
	reader := bytes.NewReader(b)

	digester := NewSha3Digester512(scheme)
	digest := digester.DigestData(reader)
	digestBytes := longbits.AsBytes(digest)

	signature, _ := signer.Sign(digestBytes)

	sig := cryptkit.NewSignature(longbits.NewBits512FromBytes(signature.Bytes()), SHA3Digest512.SignedBy(SECP256r1Sign))

	_, _ = reader.Seek(0, io.SeekStart)
	require.True(t, dv.IsValidDataSignature(reader, sig))
}

func TestECDSASignatureVerifier_IsValidDataSignature_InvalidMethod(t *testing.T) {
	dv := NewECDSASignatureVerifier(scheme, NewECDSAPublicKeyStoreFromPK(publicKey))

	signer := scheme.DigestSigner(privateKey)

	b := make([]byte, 120)
	_, _ = rand.Read(b)
	reader := bytes.NewReader(b)

	digester := NewSha3Digester512(scheme)
	digest := digester.DigestData(reader)
	digestBytes := longbits.AsBytes(digest)

	signature, _ := signer.Sign(digestBytes)

	bits := longbits.NewBits512FromBytes(signature.Bytes())

	_, _ = reader.Seek(0, io.SeekStart)
	sig1 := cryptkit.NewSignature(bits, SHA3Digest512.SignedBy(SECP256r1Sign))
	require.True(t, dv.IsValidDataSignature(reader, sig1))

	_, _ = reader.Seek(0, io.SeekStart)
	sig2 := cryptkit.NewSignature(bits, "SOME DIGEST METHOD")
	require.False(t, dv.IsValidDataSignature(reader, sig2))

	_, _ = reader.Seek(0, io.SeekStart)
	sig3 := cryptkit.NewSignature(bits, SHA3Digest512.SignedBy("SOME SIGN METHOD"))
	require.False(t, dv.IsValidDataSignature(reader, sig3))

	_, _ = reader.Seek(0, io.SeekStart)
	sig4 := cryptkit.NewSignature(bits, cryptkit.DigestMethod("SOME DIGEST METHOD").SignedBy(SECP256r1Sign))
	require.False(t, dv.IsValidDataSignature(reader, sig4))
}
