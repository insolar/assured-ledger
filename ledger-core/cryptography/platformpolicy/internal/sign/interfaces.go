// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package sign

import (
	"crypto"

	"github.com/insolar/assured-ledger/ledger-core/cryptography"
)

type AlgorithmProvider interface {
	DataSigner(crypto.PrivateKey, cryptography.Hasher) cryptography.Signer
	DigestSigner(crypto.PrivateKey) cryptography.Signer
	DataVerifier(crypto.PublicKey, cryptography.Hasher) cryptography.Verifier
	DigestVerifier(crypto.PublicKey) cryptography.Verifier
}
