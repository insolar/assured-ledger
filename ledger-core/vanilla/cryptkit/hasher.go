// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cryptkit

import (
	"hash"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/iokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ hash.Hash = DigestHasher{}

type DigestHasher struct {
	BasicDigester
	hash.Hash
}

func (v DigestHasher) DigestReader(r io.Reader) DigestHasher {
	if _, err := io.Copy(v.Hash, r); err != nil {
		panic(err)
	}
	return v
}

func (v DigestHasher) DigestOf(w io.WriterTo) DigestHasher {
	if _, err := w.WriteTo(v.Hash); err != nil {
		panic(err)
	}
	return v
}

func (v DigestHasher) DigestBytes(b []byte) DigestHasher {
	if _, err := v.Hash.Write(b); err != nil {
		panic(err)
	}
	return v
}

func (v DigestHasher) SumToDigest() Digest {
	return DigestOfHash(v.BasicDigester, v.Hash)
}

func (v DigestHasher) SumToDigestBytes(b []byte) {
	HashToBytes(v.Hash, b)
}

func (v DigestHasher) SumToSignature(signer DataSigner) Signature {
	digest := v.SumToDigest()
	return signer.SignDigest(digest)
}

func HashToBytes(hasher hash.Hash, b []byte) {
	n := len(b)
	if h := hasher.Sum(b[:0:n]); len(h) != n {
		panic(throw.IllegalValue())
	}
}

func ByteDigestOfHash(digester BasicDigester, hasher hash.Hash) []byte {
	n := digester.GetDigestSize()
	h := hasher.Sum(make([]byte, 0, n))
	if len(h) < n {
		panic(throw.IllegalValue())
	}
	return h[:n]
}

func DigestOfHash(digester BasicDigester, hasher hash.Hash) Digest {
	h := ByteDigestOfHash(digester, hasher)
	return NewDigest(longbits.WrapBytes(h), digester.GetDigestMethod())
}

func NewHashingTeeReader(hasher DigestHasher, r io.Reader) HashingTeeReader {
	tr := HashingTeeReader{}
	tr.BasicDigester = hasher.BasicDigester
	tr.CopyTo = hasher.Hash
	tr.R = r
	return tr
}

type HashingTeeReader struct {
	BasicDigester
	iokit.TeeReader
}

func (v HashingTeeReader) SumToDigest() Digest {
	return DigestOfHash(v.BasicDigester, v.CopyTo.(hash.Hash))
}

func (v *HashingTeeReader) SumToDigestAndSignature(m SigningMethod) (d Digest, s Signature, err error) {
	d = DigestOfHash(v.BasicDigester, v.CopyTo.(hash.Hash))
	s, err = v.ReadSignature(m)
	return
}

func (v *HashingTeeReader) ReadSignatureBytes() ([]byte, error) {
	b := make([]byte, v.GetDigestSize())
	n, err := io.ReadFull(&v.TeeReader, b)
	return b[:n], err
}

func (v *HashingTeeReader) ReadSignature(m SigningMethod) (Signature, error) {
	b, err := v.ReadSignatureBytes()
	if err != nil {
		return Signature{}, err
	}
	return NewSignature(longbits.WrapBytes(b), v.GetDigestMethod().SignedBy(m)), nil
}

func (v *HashingTeeReader) ReadAndVerifySignature(verifier DataSignatureVerifier) ([]byte, error) {
	d := DigestOfHash(v.BasicDigester, v.CopyTo.(hash.Hash))
	b, err := v.ReadSignatureBytes()
	if err != nil {
		return nil, err
	}
	s := NewSignature(longbits.WrapBytes(b), verifier.GetSignatureMethod())
	if !verifier.IsValidDigestSignature(d, s) {
		err = throw.RemoteBreach("packet signature mismatch")
	}
	return b, err
}
