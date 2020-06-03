// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package transport

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
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
