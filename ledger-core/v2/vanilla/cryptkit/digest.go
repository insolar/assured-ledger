// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cryptkit

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

type DigestMethod string

func (d DigestMethod) SignedBy(s SigningMethod) SignatureMethod {
	return SignatureMethod(string(d) + "/" + string(s))
}

func (d DigestMethod) String() string {
	return string(d)
}

type BasicDigester interface {
	GetDigestMethod() DigestMethod
	GetDigestSize() int
}

type DataDigester interface {
	BasicDigester
	DigestData(io.Reader) Digest
	// DigestBytes([]byte) Digest
}

type PairDigester interface {
	BasicDigester
	DigestPair(digest0 longbits.FoldableReader, digest1 longbits.FoldableReader) Digest
}

type SequenceDigester interface {
	BasicDigester
	AddNext(digest longbits.FoldableReader)
	FinishSequence() Digest
}

type ForkingDigester interface {
	SequenceDigester
	ForkSequence() ForkingDigester
}

type DigestFactory interface {
	CreatePairDigester() PairDigester
	CreateDataDigester() DataDigester
	CreateSequenceDigester() SequenceDigester
	CreateForkingDigester() ForkingDigester
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit.DigestHolder -o . -s _mock.go -g

type DigestHolder interface {
	longbits.FoldableReader
	SignWith(signer DigestSigner) SignedDigestHolder
	CopyOfDigest() Digest
	GetDigestMethod() DigestMethod
	Equals(other DigestHolder) bool
}
