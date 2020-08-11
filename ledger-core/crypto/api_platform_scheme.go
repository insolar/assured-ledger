// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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
	RefDataDigester() cryptkit.DataDigester
}

type RecordScheme interface {
	ReferenceScheme

	cryptkit.SignatureVerifierFactory
	cryptkit.KeyStoreFactory

	RecordDigester() cryptkit.DataDigester
	RecordSigner() cryptkit.DigestSigner
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
