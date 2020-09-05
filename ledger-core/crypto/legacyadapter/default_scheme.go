// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package legacyadapter

import (
	"crypto/ecdsa"

	"github.com/insolar/assured-ledger/ledger-core/crypto"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

func New(pcs cryptography.PlatformCryptographyScheme, _ cryptography.KeyProcessor, ks cryptography.KeyStore) crypto.PlatformScheme {
	localSK, err := ks.GetPrivateKey("")
	if err != nil {
		panic(err)
	}
	ecdsaSK := localSK.(*ecdsa.PrivateKey)

	return &platformSchemeLegacy{
		pcs: pcs,
		signer: NewECDSADigestSigner(NewECDSASecretKeyStore(ecdsaSK), pcs),
		verifier: NewECDSASignatureVerifier(pcs, NewECDSAPublicKeyStoreFromPK(ecdsaSK.Public())),
	}
}

var _ crypto.PlatformScheme = &platformSchemeLegacy{}
type platformSchemeLegacy struct {
	pcs cryptography.PlatformCryptographyScheme
	// kp  cryptography.KeyProcessor
	signer cryptkit.DigestSigner
	verifier cryptkit.SignatureVerifier
}

func (p *platformSchemeLegacy) PacketDigester() cryptkit.DataDigester {
	return NewSha3Digester512(p.pcs)
}

func (p *platformSchemeLegacy) PacketSigner() cryptkit.DigestSigner {
	return p.signer
}

func (p *platformSchemeLegacy) NewMerkleDigester() cryptkit.PairDigester {
	return NewSha3Digester512(p.pcs)
}

func (p *platformSchemeLegacy) ReferenceDigester() cryptkit.DataDigester {
	return p.RecordScheme().ReferenceDigester()
}

func (p *platformSchemeLegacy) CreateSignatureVerifierWithPKS(pks cryptkit.PublicKeyStore) cryptkit.SignatureVerifier {
	return NewECDSASignatureVerifier(p.pcs, pks)
}

func (p *platformSchemeLegacy) CreatePublicKeyStore(skh cryptkit.SigningKeyHolder) cryptkit.PublicKeyStore {
	return NewECDSAPublicKeyStore(skh)
}

func (p *platformSchemeLegacy) ReferenceScheme() crypto.ReferenceScheme {
	return p
}

func (p *platformSchemeLegacy) RecordScheme() crypto.RecordScheme {
	return p.CustomScheme("")
}

func (p *platformSchemeLegacy) TransportScheme() crypto.TransportScheme {
	return p
}

func (p *platformSchemeLegacy) ConsensusScheme() crypto.ConsensusScheme {
	return p
}

func (p *platformSchemeLegacy) CustomScheme(name crypto.SchemeName) crypto.CustomScheme {
	if name == "" || name == crypto.PlatformSchemeName {
		return recordSchemeLegacy{p.pcs, p.signer, p.verifier }
	}
	return nil
}

