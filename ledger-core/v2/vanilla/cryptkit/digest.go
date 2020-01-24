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

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/common/cryptkit.DigestHolder -o . -s _mock.go -g

type DigestHolder interface {
	longbits.FoldableReader
	SignWith(signer DigestSigner) SignedDigestHolder
	CopyOfDigest() Digest
	GetDigestMethod() DigestMethod
	Equals(other DigestHolder) bool
}
