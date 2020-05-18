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

// deprecated
// Generate reference from hash code.
func GenerateProtoReferenceFromCode(pulse insolar.PulseNumber, code []byte) reference.Global {
	hasher := platformpolicy.NewPlatformCryptographyScheme().ReferenceHasher()
	codeHash := reference.BytesToLocalHash(hasher.Hash(code))
	id := reference.NewRecordID(pulse, codeHash)
	return reference.NewSelf(id)
}

// deprecated
// Generate prototype reference from contract id.
func GenerateProtoReferenceFromContractID(typeContractID string, name string, version int) reference.Global {
	contractID := fmt.Sprintf("%s::%s::v%02d", typeContractID, name, version)
	return GenerateProtoReferenceFromCode(pulse.BuiltinContract, []byte(contractID))
}

// deprecated
// Generate contract reference from contract id.
func GenerateCodeReferenceFromContractID(typeContractID string, name string, version int) reference.Global {
	contractID := fmt.Sprintf("%s::%s::v%02d", typeContractID, name, version)
	hasher := platformpolicy.NewPlatformCryptographyScheme().ReferenceHasher()
	codeHash := reference.BytesToLocalHash(hasher.Hash([]byte(contractID)))
	id := reference.NewRecordID(pulse.BuiltinContract, codeHash)
	return reference.NewRecordRef(id)
}

// deprecated
func GenesisRef(s string) reference.Global {
	hasher := platformpolicy.NewPlatformCryptographyScheme().ReferenceHasher()
	hash := hasher.Hash([]byte(s))
	local := reference.NewLocal(pulse.MinTimePulse, 0, reference.BytesToLocalHash(hash))
	return reference.NewSelf(local)
}
