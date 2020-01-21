//
//    Copyright 2019 Insolar Technologies
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
//

package cryptkit

import (
	"io"
	"strings"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

type SigningMethod string

func (s SigningMethod) String() string {
	return string(s)
}

type SignatureMethod string /* Digest + Sign methods */

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

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/common/cryptkit.SignatureHolder -o . -s _mock.go -g

type SignatureHolder interface {
	longbits.FoldableReader
	CopyOfSignature() Signature
	GetSignatureMethod() SignatureMethod
	Equals(other SignatureHolder) bool
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/common/cryptkit.SignatureKeyHolder -o . -s _mock.go -g

type SignatureKeyHolder interface {
	longbits.FoldableReader
	GetSigningMethod() SigningMethod
	GetSignatureKeyMethod() SignatureMethod
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

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/common/cryptkit.DigestSigner -o . -s _mock.go -g

type DigestSigner interface {
	SignDigest(digest Digest) Signature
	GetSigningMethod() SigningMethod
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/common/cryptkit.DataSigner -o . -s _mock.go -g

type DataSigner interface {
	DigestSigner
	DataDigester
	SignData(reader io.Reader) SignedDigest
	GetSignatureMethod() SignatureMethod
}

type SequenceSigner interface {
	DigestSigner
	NewSequenceDigester() SequenceDigester
	GetSignatureMethod() SignatureMethod
}

type SignedEvidenceHolder interface {
	GetEvidence() SignedData
}

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

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/common/cryptkit.SignatureVerifier -o . -s _mock.go -g

type SignatureVerifier interface {
	IsDigestMethodSupported(m DigestMethod) bool
	IsSignMethodSupported(m SigningMethod) bool
	IsSignOfSignatureMethodSupported(m SignatureMethod) bool

	IsValidDigestSignature(digest DigestHolder, signature SignatureHolder) bool
	IsValidDataSignature(data io.Reader, signature SignatureHolder) bool
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/common/cryptkit.SignatureVerifierFactory -o . -s _mock.go -g

type SignatureVerifierFactory interface {
	CreateSignatureVerifierWithPKS(pks PublicKeyStore) SignatureVerifier
}
