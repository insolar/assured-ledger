// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package adapters

import (
	"crypto/ecdsa"

	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/consensus/adapters/candidate"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

type Candidate candidate.Profile

func (c Candidate) StaticProfile(keyProcessor cryptography.KeyProcessor) *StaticProfile {
	publicKey, err := keyProcessor.ImportPublicKeyBinary(c.PublicKey)
	if err != nil {
		panic("Failed to import public key")
	}

	signHolder := cryptkit.NewSignature(
		longbits.NewBits512FromBytes(c.Signature),
		SHA3512Digest.SignedBy(SECP256r1Sign),
	).AsSignatureHolder()

	extension := newStaticProfileExtension(
		c.ShortID,
		c.Ref,
		signHolder,
	)

	return newStaticProfile(
		c.ShortID,
		c.PrimaryRole,
		c.SpecialRole,
		extension,
		NewOutbound(c.Address),
		NewECDSAPublicKeyStore(publicKey.(*ecdsa.PublicKey)),
		NewECDSASignatureKeyHolder(publicKey.(*ecdsa.PublicKey), keyProcessor),
		cryptkit.NewSignedDigest(
			cryptkit.NewDigest(longbits.NewBits512FromBytes(c.Digest), SHA3512Digest),
			cryptkit.NewSignature(longbits.NewBits512FromBytes(c.Signature), SHA3512Digest.SignedBy(SECP256r1Sign)),
		).AsSignedDigestHolder(),
	)
}

func (c Candidate) Profile() candidate.Profile {
	return candidate.Profile(c)
}

func NewCandidate(staticProfile *StaticProfile, keyProcessor cryptography.KeyProcessor) *Candidate {
	pubKey, err := keyProcessor.ExportPublicKeyBinary(staticProfile.store.(*ECDSAPublicKeyStore).publicKey)
	if err != nil {
		panic("failed to export public key")
	}

	signedDigest := staticProfile.GetBriefIntroSignedDigest()

	return &Candidate{
		Address:     staticProfile.GetDefaultEndpoint().GetIPAddress().String(),
		Ref:         staticProfile.GetExtension().GetReference(),
		ShortID:     staticProfile.GetStaticNodeID(),
		PrimaryRole: staticProfile.GetPrimaryRole(),
		SpecialRole: staticProfile.GetSpecialRoles(),
		Digest:      longbits.AsBytes(signedDigest.GetDigestHolder()),
		Signature:   longbits.AsBytes(signedDigest.GetSignatureHolder()),
		PublicKey:   pubKey,
	}
}
