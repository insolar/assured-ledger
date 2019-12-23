package api

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/platformpolicy"
)

func MakeReason(pulse insolar.PulseNumber, data []byte) insolar.Reference {
	hasher := platformpolicy.NewPlatformCryptographyScheme().ReferenceHasher()
	reasonID := *insolar.NewID(pulse, hasher.Hash(data))
	return *insolar.NewRecordReference(reasonID)
}
