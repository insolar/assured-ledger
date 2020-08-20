// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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

func (s SignatureMethod) SignMethod() SigningMethod {
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
	CopyOfSignature() Signature
	GetSignatureMethod() SignatureMethod
	Equals(other SignatureHolder) bool
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit.SignatureKeyHolder -o . -s _mock.go -g

// TODO rename to SigningKeyHolder
type SignatureKeyHolder interface {
	longbits.FoldableReader
	GetSigningMethod() SigningMethod
	GetSignatureKeyMethod() SignatureMethod
	// TODO rename to GetSigningKeyType
	GetSignatureKeyType() SignatureKeyType
	Equals(other SignatureKeyHolder) bool
}

type SignedDigestHolder interface {
	CopyOfSignedDigest() SignedDigest
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

// TODO rename to SigningKeyType
type SignatureKeyType uint8

const (
	SymmetricKey SignatureKeyType = iota
	SecretAsymmetricKey
	PublicAsymmetricKey
)

func (v SignatureKeyType) IsSymmetric() bool {
	return v == SymmetricKey
}

func (v SignatureKeyType) IsSecret() bool {
	return v != PublicAsymmetricKey
}

type DataSignatureVerifier interface {
	DataDigester
	GetDefaultSignatureMethod() SignatureMethod
	SignatureVerifier
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit.SignatureVerifier -o . -s _mock.go -g

type SignatureVerifier interface {
	IsDigestMethodSupported(m DigestMethod) bool
	IsSignMethodSupported(m SigningMethod) bool
	IsSignOfSignatureMethodSupported(m SignatureMethod) bool

	IsValidDigestSignature(digest DigestHolder, signature SignatureHolder) bool
	IsValidDataSignature(data io.Reader, signature SignatureHolder) bool
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit.SignatureVerifierFactory -o . -s _mock.go -g

type SignatureVerifierFactory interface {
	CreateSignatureVerifierWithPKS(PublicKeyStore) SignatureVerifier
	// CreateSignatureVerifierWithKey(SignatureKeyHolder) SignatureVerifier
	// TODO Add	CreateDataSignatureVerifier(k SignatureKey, m SignatureMethod) DataSignatureVerifier
}

type DataSignatureVerifierFactory interface {
	IsSignatureKeySupported(SignatureKey) bool
	CreateDataSignatureVerifier(SignatureKey) DataSignatureVerifier
}

type DataSignerFactory interface {
	IsSignatureKeySupported(SignatureKey) bool
	CreateDataSigner(SignatureKey) DataSigner
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

