package cryptkit

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
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

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit.DataDigester -o . -s _mock.go -g

type DataDigester interface {
	BasicDigester
	// deprecated
	DigestData(io.Reader) Digest
	DigestBytes([]byte) Digest
	NewHasher() DigestHasher
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit.PairDigester -o . -s _mock.go -g

type PairDigester interface {
	BasicDigester
	DigestPair(digest0 longbits.FoldableReader, digest1 longbits.FoldableReader) Digest
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit.SequenceDigester -o . -s _mock.go -g

type SequenceDigester interface {
	BasicDigester
	AddNext(digest longbits.FoldableReader)
	FinishSequence() Digest
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit.ForkingDigester -o . -s _mock.go -g

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

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit.DigestHolder -o . -s _mock.go -g

type DigestHolder interface {
	longbits.FoldableReader
	SignWith(signer DigestSigner) SignedDigestHolder
	GetDigestMethod() DigestMethod
	Equals(other DigestHolder) bool
}
