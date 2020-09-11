// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package legacyadapter

import (
	"github.com/insolar/assured-ledger/ledger-core/crypto"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type recordSchemeLegacy struct {
	pcs cryptography.PlatformCryptographyScheme
	signer cryptkit.DigestSigner
	verifier cryptkit.SignatureVerifier
}

func (v recordSchemeLegacy) ReferenceDigester() cryptkit.DataDigester {
	return NewSha3Digester512as224(v.pcs)
}

func (v recordSchemeLegacy) CreateSignatureVerifierWithPKS(pks cryptkit.PublicKeyStore) cryptkit.SignatureVerifier {
	return NewECDSASignatureVerifier(v.pcs, pks)
}

func (v recordSchemeLegacy) CreatePublicKeyStore(skh cryptkit.SigningKeyHolder) cryptkit.PublicKeyStore {
	return NewECDSAPublicKeyStore(skh)
}

func (v recordSchemeLegacy) RecordDigester() crypto.RecordDigester {
	return recordDualDigester{NewSha3Digester512(v.pcs) }
}

func (v recordSchemeLegacy) RecordSigner() cryptkit.DigestSigner {
	return v.signer
}

func (v recordSchemeLegacy) SelfVerifier() cryptkit.SignatureVerifier {
	return v.verifier
}

func (v recordSchemeLegacy) GetSchemeName() crypto.SchemeName {
	return crypto.PlatformSchemeName
}

