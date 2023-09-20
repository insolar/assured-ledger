package transport

import (
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

type CryptographyAssistant interface {
	cryptkit.SignatureVerifierFactory
	cryptkit.KeyStoreFactory
	GetDigestFactory() ConsensusDigestFactory
	CreateNodeSigner(sks cryptkit.SecretKeyStore) cryptkit.DigestSigner
}

type ConsensusDigestFactory interface {
	cryptkit.DigestFactory
	CreateAnnouncementDigester() cryptkit.ForkingDigester
	CreateGlobulaStateDigester() StateDigester
}

type StateDigester interface {
	AddNext(digest longbits.FoldableReader, fullRank member.FullRank)
	GetDigestMethod() cryptkit.DigestMethod
	/* deprecated */
	ForkSequence() StateDigester

	FinishSequence() cryptkit.Digest
}
