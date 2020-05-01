// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package genesisrefs

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

const (
	PrototypeType    = "prototype"
	PrototypeSuffix  = "_proto"
	FundsDepositName = "genesis_deposit"
)

// Generate reference from hash code.
// deprecated
func GenerateProtoReferenceFromCode(pulse insolar.PulseNumber, code []byte) insolar.Reference {
	hasher := platformpolicy.NewPlatformCryptographyScheme().ReferenceHasher()
	codeHash := hasher.Hash(code)
	id := insolar.NewID(pulse, codeHash)
	return insolar.NewReference(id)
}

// Generate prototype reference from contract id.
// deprecated
func GenerateProtoReferenceFromContractID(typeContractID string, name string, version int) insolar.Reference {
	contractID := fmt.Sprintf("%s::%s::v%02d", typeContractID, name, version)
	return GenerateProtoReferenceFromCode(pulse.BuiltinContract, []byte(contractID))
}

// Generate contract reference from contract id.
// deprecated
func GenerateCodeReferenceFromContractID(typeContractID string, name string, version int) insolar.Reference {
	contractID := fmt.Sprintf("%s::%s::v%02d", typeContractID, name, version)
	hasher := platformpolicy.NewPlatformCryptographyScheme().ReferenceHasher()
	codeHash := hasher.Hash([]byte(contractID))
	id := insolar.NewID(pulse.BuiltinContract, codeHash)
	return insolar.NewRecordReference(id)
}

// deprecated
func GenesisRef(s string) insolar.Reference {
	hasher := platformpolicy.NewPlatformCryptographyScheme().ReferenceHasher()
	hash := hasher.Hash([]byte(s))
	local := reference.NewLocal(pulse.MinTimePulse, 0, reference.BytesToLocalHash(hash))
	return reference.NewGlobal(local, local)
}
