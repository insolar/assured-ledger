package testutils

import (
	"github.com/insolar/assured-ledger/ledger-core/crypto"
	"github.com/insolar/assured-ledger/ledger-core/crypto/legacyadapter"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/vnlmn"
)

// func GetReferenceBuilder(nodeRef reference.Holder) vnlmn.RecordReferenceBuilder {
// 	if nodeRef.IsEmpty() {
// 		panic(throw.IllegalValue())
// 	}
//
// 	keyProcessor := platformpolicy.NewKeyProcessor()
// 	pk, err := keyProcessor.GeneratePrivateKey()
// 	if err != nil {
// 		panic(throw.W(err, "failed to generate node PK"))
// 	}
// 	keyStore := keystore.NewInplaceKeyStore(pk)
//
// 	platformCryptographyScheme := platformpolicy.NewPlatformCryptographyScheme()
// 	platformScheme := legacyadapter.New(platformCryptographyScheme, keyProcessor, keyStore)
//
// 	return vnlmn.NewRecordReferenceBuilder(platformScheme.RecordScheme(), nodeRef)
// }

func GetPCS(nodeRef reference.Holder) crypto.PlatformScheme {
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

	return platformScheme
}

func GetDataDigester(nodeRef reference.Holder) cryptkit.DataDigester {
	return GetPCS(nodeRef).RecordScheme().ReferenceDigester()
}

func GetObjectReference(request *rms.VCallRequest, nodeRef reference.Holder) reference.Global {
	return vnlmn.GetLifelineAnticipatedReference(GetDataDigester(nodeRef), request, pulse.Unknown)
}
