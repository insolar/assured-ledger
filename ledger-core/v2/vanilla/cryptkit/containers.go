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
	"fmt"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

type hFoldReader = longbits.FoldableReader

func NewDigest(data longbits.FoldableReader, method DigestMethod) Digest {
	return Digest{hFoldReader: data, digestMethod: method}
}

type Digest struct {
	hFoldReader
	digestMethod DigestMethod
}

func (d Digest) IsEmpty() bool {
	return d.hFoldReader == nil
}

func (d Digest) CopyOfDigest() Digest {
	return Digest{hFoldReader: longbits.CopyToMutable(d.hFoldReader), digestMethod: d.digestMethod}
}

func (d Digest) Equals(o DigestHolder) bool {
	return longbits.EqualFixedLenWriterTo(d, o)
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

func (p Signature) CopyOfSignature() Signature {
	return Signature{hFoldReader: longbits.CopyToMutable(p.hFoldReader), signatureMethod: p.signatureMethod}
}

func (p Signature) Equals(o SignatureHolder) bool {
	return longbits.EqualFixedLenWriterTo(p, o)
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

func (r SignedDigest) CopyOfSignedDigest() SignedDigest {
	return NewSignedDigest(r.digest.CopyOfDigest(), r.signature.CopyOfSignature())
}

func (r SignedDigest) Equals(o SignedDigestHolder) bool {
	return longbits.EqualFixedLenWriterTo(r.digest, o.GetDigestHolder()) &&
		longbits.EqualFixedLenWriterTo(r.signature, o.GetSignatureHolder())
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
	return v.IsSignOfSignatureMethodSupported(r.signature.GetSignatureMethod())
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

func NewSignedData(data io.Reader, digest Digest, signature Signature) SignedData {
	return SignedData{SignedDigest{digest, signature}, data}
}

func SignDataByDataSigner(data io.Reader, signer DataSigner) SignedData {
	sd := signer.SignData(data)
	return NewSignedData(data, sd.digest, sd.signature)
}

type hReader = io.Reader
type hSignedDigest = SignedDigest

var _ io.WriterTo = SignedData{}

type SignedData struct {
	hSignedDigest
	hReader
}

func (r SignedData) IsEmpty() bool {
	return r.hReader == nil && r.hSignedDigest.IsEmpty()
}

func (r SignedData) GetSignedDigest() SignedDigest {
	return r.hSignedDigest
}

func (r SignedData) WriteTo(w io.Writer) (int64, error) {
	return io.Copy(w, r.hReader)
}

func (r SignedData) String() string {
	return fmt.Sprintf("[bytes=%v]%v", r.hReader, r.hSignedDigest)
}

/*****************************************************************/

func NewSignatureKey(data longbits.FoldableReader, signatureMethod SignatureMethod, keyType SignatureKeyType) SignatureKey {
	return SignatureKey{
		hFoldReader:     data,
		signatureMethod: signatureMethod,
		keyType:         keyType,
	}
}

var _ SignatureKeyHolder = SignatureKey{}

type SignatureKey struct {
	hFoldReader
	signatureMethod SignatureMethod
	keyType         SignatureKeyType
}

func (p SignatureKey) IsEmpty() bool {
	return p.hFoldReader == nil
}

func (p SignatureKey) GetSigningMethod() SigningMethod {
	return p.signatureMethod.SignMethod()
}

func (p SignatureKey) GetSignatureKeyMethod() SignatureMethod {
	return p.signatureMethod
}

func (p SignatureKey) GetSignatureKeyType() SignatureKeyType {
	return p.keyType
}

func (p SignatureKey) Equals(o SignatureKeyHolder) bool {
	return longbits.EqualFixedLenWriterTo(p, o)
}

func (p SignatureKey) String() string {
	return fmt.Sprintf("⚿%v", p.hFoldReader)
}
