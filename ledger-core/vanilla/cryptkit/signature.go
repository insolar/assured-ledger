package cryptkit

import (
	"io"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type SigningMethod string

func (s SigningMethod) String() string {
	return string(s)
}

type SignatureMethod string /* Digest + Signing methods */

func (s SignatureMethod) DigestMethod() DigestMethod {
	parts := strings.Split(string(s), "/")
	if len(parts) != 2 {
		return ""
	}
	return DigestMethod(parts[0])
}

func (s SignatureMethod) SigningMethod() SigningMethod {
	parts := strings.Split(string(s), "/")
	if len(parts) != 2 {
		return ""
	}
	return SigningMethod(parts[1])
}

func (s SignatureMethod) String() string {
	return string(s)
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit.SignatureHolder -o . -s _mock.go -g

type SignatureHolder interface {
	longbits.FoldableReader
	GetSignatureMethod() SignatureMethod
	Equals(other SignatureHolder) bool
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit.SigningKeyHolder -o . -s _mock.go -g

type SigningKeyHolder interface {
	longbits.FoldableReader
	GetSigningMethod() SigningMethod
	GetSigningKeyType() SigningKeyType
	Equals(other SigningKeyHolder) bool
}

type SignedDigestHolder interface {
	Equals(o SignedDigestHolder) bool
	GetDigestHolder() DigestHolder
	GetSignatureHolder() SignatureHolder
	GetSignatureMethod() SignatureMethod
	IsVerifiableBy(v SignatureVerifier) bool
	VerifyWith(v SignatureVerifier) bool
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit.DigestSigner -o . -s _mock.go -g

type DigestSigner interface {
	SignDigest(digest Digest) Signature
	GetSigningMethod() SigningMethod
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit.DataSigner -o . -s _mock.go -g

type DataSigner interface {
	DigestSigner
	DataDigester
	GetSignatureMethod() SignatureMethod
}

type SequenceSigner interface {
	DigestSigner
	NewSequenceDigester() SequenceDigester
	// GetSignatureMethod() SignatureMethod
}

type SignedEvidenceHolder interface {
	GetEvidence() SignedData
}

type SigningKeyType uint8

const (
	SymmetricKey SigningKeyType = iota
	SecretAsymmetricKey
	PublicAsymmetricKey
)

func (v SigningKeyType) IsSymmetric() bool {
	return v == SymmetricKey
}

func (v SigningKeyType) IsSecret() bool {
	return v != PublicAsymmetricKey
}

type DataSignatureVerifier interface {
	DataDigester
	GetDefaultSignatureMethod() SignatureMethod
	SignatureVerifier
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit.SignatureVerifier -o . -s _mock.go -g

type SignatureVerifier interface {
	GetDefaultSigningMethod() SigningMethod

	IsDigestMethodSupported(m DigestMethod) bool
	IsSigningMethodSupported(m SigningMethod) bool

	IsValidDigestSignature(digest DigestHolder, signature SignatureHolder) bool
	IsValidDataSignature(data io.Reader, signature SignatureHolder) bool
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit.SignatureVerifierFactory -o . -s _mock.go -g

type SignatureVerifierFactory interface {
	CreateSignatureVerifierWithPKS(PublicKeyStore) SignatureVerifier
	// CreateSignatureVerifierWithKey(SigningKeyHolder) SignatureVerifier
	// TODO Add	CreateDataSignatureVerifier(k SigningKey, m SignatureMethod) DataSignatureVerifier
}

type DataSignatureVerifierFactory interface {
	IsSignatureKeySupported(SigningKey) bool
	CreateDataSignatureVerifier(SigningKey) DataSignatureVerifier
}

type DataSignerFactory interface {
	IsSignatureKeySupported(SigningKey) bool
	CreateDataSigner(SigningKey) DataSigner
}



type dataSigner struct {
	DigestSigner
	DataDigester
}

func (v dataSigner) GetSignatureMethod() SignatureMethod {
	return v.DataDigester.GetDigestMethod().SignedBy(v.DigestSigner.GetSigningMethod())
}

func AsDataSigner(dd DataDigester, ds DigestSigner) DataSigner {
	switch {
	case ds == nil:
		panic(throw.IllegalValue())
	case dd == nil:
		panic(throw.IllegalValue())
	}

	return dataSigner{ds, dd}
}

func AsDataSignatureVerifier(dd DataDigester, sv SignatureVerifier, defSigning SigningMethod) DataSignatureVerifier {
	switch {
	case sv == nil:
		panic(throw.IllegalValue())
	case dd == nil:
		panic(throw.IllegalValue())
	}

	return dataSignatureVerifier{dd, sv, dd.GetDigestMethod().SignedBy(defSigning) }
}

type dataSignatureVerifier struct {
	DataDigester
	SignatureVerifier

	defSignature SignatureMethod
}

func (v dataSignatureVerifier) GetDefaultSignatureMethod() SignatureMethod {
	return v.defSignature
}

