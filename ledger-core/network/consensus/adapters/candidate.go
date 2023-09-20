package adapters

import (
	"crypto/ecdsa"

	"github.com/insolar/assured-ledger/ledger-core/crypto/legacyadapter"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

type Candidate rms.Profile

func (c Candidate) StaticProfile(keyProcessor cryptography.KeyProcessor) *StaticProfile {
	publicKey, err := keyProcessor.ImportPublicKeyBinary(c.PublicKey)
	if err != nil {
		panic("Failed to import public key")
	}

	signHolder := cryptkit.NewSignature(
		longbits.NewBits512FromBytes(c.Signature),
		legacyadapter.SHA3Digest512.SignedBy(legacyadapter.SECP256r1Sign),
	)

	extension := NewStaticProfileExtension(
		c.ShortID,
		c.Ref.GetValue(),
		signHolder,
	)

	// TODO start power level is not passed properly - needs fix
	startPower := DefaultStartPower

	return NewStaticProfile(c.ShortID, c.PrimaryRole, c.SpecialRole, startPower,
		extension,
		NewOutbound(c.Address),
		legacyadapter.NewECDSAPublicKeyStoreFromPK(publicKey),
		legacyadapter.NewECDSASignatureKeyHolder(publicKey.(*ecdsa.PublicKey), keyProcessor),
		cryptkit.NewSignedDigest(
			cryptkit.NewDigest(longbits.NewBits512FromBytes(c.Digest), legacyadapter.SHA3Digest512),
			cryptkit.NewSignature(longbits.NewBits512FromBytes(c.Signature), legacyadapter.SHA3Digest512.SignedBy(legacyadapter.SECP256r1Sign)),
		),
	)
}

func (c Candidate) Profile() rms.Profile {
	return rms.Profile(c)
}

func NewCandidate(staticProfile *StaticProfile, keyProcessor cryptography.KeyProcessor) *Candidate {
	pubKey, err := keyProcessor.ExportPublicKeyBinary(
		staticProfile.store.(*legacyadapter.ECDSAPublicKeyStore).CryptoPublicKey())

	if err != nil {
		panic("failed to export public key")
	}

	signedDigest := staticProfile.GetBriefIntroSignedDigest()

	return &Candidate{
		Address:     staticProfile.GetDefaultEndpoint().GetIPAddress().String(),
		Ref:         rms.NewReference(staticProfile.GetExtension().GetReference()),
		ShortID:     staticProfile.GetStaticNodeID(),
		PrimaryRole: staticProfile.GetPrimaryRole(),
		SpecialRole: staticProfile.GetSpecialRoles(),
		Digest:      longbits.AsBytes(signedDigest.GetDigestHolder()),
		Signature:   longbits.AsBytes(signedDigest.GetSignatureHolder()),
		PublicKey:   pubKey,
	}
}
