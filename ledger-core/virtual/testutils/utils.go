// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testutils

import (
	"github.com/insolar/assured-ledger/ledger-core/crypto/legacyadapter"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/vnlmn"
)

func GetReferenceBuilder(nodeRef reference.Holder) vnlmn.RecordReferenceBuilder {
	if nodeRef.IsEmpty() {
		panic(throw.IllegalValue())
	}

	keyProcessor := platformpolicy.NewKeyProcessor()
	pk, err := keyProcessor.GeneratePrivateKey()
	if err != nil {
		panic(throw.W(err, "failed to generate node PK"))
	}
	keyStore := keystore.NewInplaceKeyStore(pk)

	platformCryptographyScheme := platformpolicy.NewPlatformCryptographyScheme()
	platformScheme := legacyadapter.New(platformCryptographyScheme, keyProcessor, keyStore)

	return vnlmn.NewRecordReferenceBuilder(platformScheme.RecordScheme(), nodeRef)
}

func GetObjectReference(request *rms.VCallRequest, nodeRef reference.Holder) reference.Global {
	return vnlmn.GetLifelineAnticipatedReference(GetReferenceBuilder(nodeRef), request, pulse.Unknown)
}
