package crypto

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type SchemeName string
const PlatformSchemeName SchemeName = "INSv2"

type PlatformScheme interface {
	ReferenceScheme() ReferenceScheme
	RecordScheme() RecordScheme
	TransportScheme() TransportScheme
	ConsensusScheme() ConsensusScheme

	CustomScheme(SchemeName) CustomScheme
}

type CustomScheme interface {
	RecordScheme
	GetSchemeName() SchemeName
}

type ReferenceScheme interface {
	ReferenceDigester() cryptkit.DataDigester
}

type RecordScheme interface {
	ReferenceScheme

	cryptkit.SignatureVerifierFactory
	cryptkit.KeyStoreFactory

	RecordDigester() RecordDigester
	RecordSigner() cryptkit.DigestSigner

	SelfVerifier() cryptkit.SignatureVerifier
}

type RecordDigester interface {
	cryptkit.DataDigester

	NewDataAndRefHasher() cryptkit.DigestHasher
	GetDataAndRefDigests(cryptkit.DigestHasher) (data, ref cryptkit.Digest)
	GetRefDigestAndContinueData(cryptkit.DigestHasher) (data cryptkit.DigestHasher, ref cryptkit.Digest)
}

type TransportScheme interface {
	cryptkit.SignatureVerifierFactory
	cryptkit.KeyStoreFactory

	PacketDigester() cryptkit.DataDigester
	PacketSigner() cryptkit.DigestSigner
}

type ConsensusScheme interface {
	cryptkit.SignatureVerifierFactory
	cryptkit.KeyStoreFactory

	PacketDigester() cryptkit.DataDigester
	PacketSigner() cryptkit.DigestSigner

	NewMerkleDigester() cryptkit.PairDigester
//	NewAnnouncementDigester() cryptkit.ForkingDigester
}
