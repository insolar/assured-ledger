// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
)

type PeerCryptographyFactory interface {
	// TODO for some reason linter can't handle multiple declarations of the same method while it is valid for 1.14
	// cryptkit.DataSignatureVerifierFactory
	// cryptkit.DataSignerFactory
	CreateDataSignatureVerifier(cryptkit.SignatureKey) cryptkit.DataSignatureVerifier
	CreateDataSigner(cryptkit.SignatureKey) cryptkit.DataSigner
	IsSignatureKeySupported(cryptkit.SignatureKey) bool
	CreateDataDecrypter(cryptkit.SignatureKey) cryptkit.Decrypter
	CreateDataEncrypter(cryptkit.SignatureKey) cryptkit.Encrypter
	GetMaxSignatureSize() int
}

