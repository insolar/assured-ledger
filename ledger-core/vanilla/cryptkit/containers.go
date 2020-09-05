// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cryptkit

import (
	"fmt"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

type hFoldReader = longbits.FoldableReader

func NewDigest(data longbits.FoldableReader, method DigestMethod) Digest {
	return Digest{hFoldReader: data, digestMethod: method}
}

func NewZeroSizeDigest(method DigestMethod) Digest {
	return Digest{hFoldReader: longbits.EmptyByteString, digestMethod: method}
}

var _ DigestHolder = Digest{}

type Digest struct {
	hFoldReader
	digestMethod DigestMethod
}

func (d Digest) IsZero() bool {
	return d.hFoldReader == nil
}

// TODO move users to IsZero and use IsEmpty for zero length, not zero state
func (d Digest) IsEmpty() bool {
	return d.hFoldReader == nil
}

func (d Digest) FixedByteSize() int {
	if d.hFoldReader != nil {
		return d.hFoldReader.FixedByteSize()
	}
	return 0
}

func (d Digest) Equals(o DigestHolder) bool {
	return longbits.Equal(d, o)
}

func (d Digest) AsDigestHolder() DigestHolder {
	if d.IsEmpty() {
		return nil
	}
	return d
}

func (d Digest) GetDigestMethod() DigestMethod {
	return d.digestMethod
}

func (d Digest) SignWith(signer DigestSigner) SignedDigestHolder {
	sd := NewSignedDigest(d, signer.SignDigest(d))
	return sd
}

func (d Digest) String() string {
	return fmt.Sprintf("%v", d.hFoldReader)
}

/*****************************************************************/

func NewSignature(data longbits.FoldableReader, method SignatureMethod) Signature {
	return Signature{hFoldReader: data, signatureMethod: method}
}

type Signature struct {
	hFoldReader
	signatureMethod SignatureMethod
}

func (p Signature) IsEmpty() bool {
	return p.hFoldReader == nil
}

func (p Signature) FixedByteSize() int {
	if p.hFoldReader != nil {
		return p.hFoldReader.FixedByteSize()
	}
	return 0
}

func (p Signature) CopyOfSignature() Signature {
	return Signature{hFoldReader: longbits.CopyFixed(p.hFoldReader), signatureMethod: p.signatureMethod}
}

func (p Signature) Equals(o SignatureHolder) bool {
	return longbits.Equal(p, o)
}

func (p Signature) GetSignatureMethod() SignatureMethod {
	return p.signatureMethod
}

func (p Signature) AsSignatureHolder() SignatureHolder {
	if p.IsEmpty() {
		return nil
	}
	return p
}

func (p Signature) String() string {
	return fmt.Sprintf("§%v", p.hFoldReader)
}

/*****************************************************************/

func NewSignedDigest(digest Digest, signature Signature) SignedDigest {
	return SignedDigest{digest: digest, signature: signature}
}

type SignedDigest struct {
	digest    Digest
	signature Signature
}

func (r SignedDigest) IsEmpty() bool {
	return r.digest.IsEmpty() && r.signature.IsEmpty()
}

func (r SignedDigest) Equals(o SignedDigestHolder) bool {
	return longbits.Equal(r.digest, o.GetDigestHolder()) &&
		longbits.Equal(r.signature, o.GetSignatureHolder())
}

func (r SignedDigest) GetDigest() Digest {
	return r.digest
}

func (r SignedDigest) GetSignature() Signature {
	return r.signature
}

func (r SignedDigest) GetDigestHolder() DigestHolder {
	return r.digest
}

func (r SignedDigest) GetSignatureHolder() SignatureHolder {
	return r.signature
}

func (r SignedDigest) GetSignatureMethod() SignatureMethod {
	return r.signature.GetSignatureMethod()
}

func (r SignedDigest) IsVerifiableBy(v SignatureVerifier) bool {
	return v.IsSigningMethodSupported(r.signature.GetSignatureMethod().SigningMethod())
}

func (r SignedDigest) VerifyWith(v SignatureVerifier) bool {
	return v.IsValidDigestSignature(r.digest, r.signature)
}

func (r SignedDigest) String() string {
	return fmt.Sprintf("%v%v", r.digest, r.signature)
}

func (r SignedDigest) AsSignedDigestHolder() SignedDigestHolder {
	if r.IsEmpty() {
		return nil
	}
	return r
}

/*****************************************************************/

func NewSignedData(data longbits.FixedReader, digest Digest, signature Signature) SignedData {
	return SignedData{SignedDigest{digest, signature}, data}
}

func SignDataByDataSigner(data longbits.FixedReader, signer DataSigner) SignedData {
	hasher := signer.NewHasher()
	if _, err := data.WriteTo(hasher); err != nil {
		panic(err)
	}
	digest := hasher.SumToDigest()
	signature := signer.SignDigest(digest)
	return NewSignedData(data, digest, signature)
}

type hWriterTo = longbits.FixedReader
type hSignedDigest = SignedDigest

var _ io.WriterTo = SignedData{}

type SignedData struct {
	hSignedDigest
	hWriterTo
}

func (r SignedData) IsEmpty() bool {
	return r.hWriterTo == nil && r.hSignedDigest.IsEmpty()
}

func (r SignedData) FixedByteSize() int {
	if r.hWriterTo != nil {
		return r.hWriterTo.FixedByteSize()
	}
	return 0
}

func (r SignedData) GetSignedDigest() SignedDigest {
	return r.hSignedDigest
}

func (r SignedData) String() string {
	return fmt.Sprintf("[bytes=%v]%v", r.hWriterTo, r.hSignedDigest)
}

/*****************************************************************/

func NewSigningKey(data longbits.FoldableReader, method SigningMethod, keyType SigningKeyType) SigningKey {
	return SigningKey{
		hFoldReader: data,
		method:      method,
		keyType:     keyType,
	}
}

var _ SigningKeyHolder = SigningKey{}

type SigningKey struct {
	hFoldReader
	method SigningMethod
	keyType         SigningKeyType
}

func (p SigningKey) IsEmpty() bool {
	return p.hFoldReader == nil
}

func (p SigningKey) GetSigningMethod() SigningMethod {
	return p.method
}

func (p SigningKey) GetSigningKeyType() SigningKeyType {
	return p.keyType
}

func (p SigningKey) FixedByteSize() int {
	if p.hFoldReader != nil {
		return p.hFoldReader.FixedByteSize()
	}
	return 0
}

func (p SigningKey) Equals(o SigningKeyHolder) bool {
	return longbits.Equal(p, o)
}

func (p SigningKey) String() string {
	return fmt.Sprintf("⚿%v", p.hFoldReader)
}
