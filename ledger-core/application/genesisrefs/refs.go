// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package genesisrefs

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/cryptography/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

const (
	ClassType        = "class"
	ClassSuffix      = "_class"
	FundsDepositName = "genesis_deposit"
)

// deprecated
// Generate class reference from hash code.
func GenerateClassReferenceFromCode(pulse pulse.Number, code []byte) reference.Global {
	hasher := platformpolicy.NewPlatformCryptographyScheme().ReferenceHasher()
	codeHash := reference.BytesToLocalHash(hasher.Hash(code))
	id := reference.NewRecordID(pulse, codeHash)
	return reference.NewSelf(id)
}

// deprecated
// Generate class reference from contract id.
func GenerateClassReferenceFromContractID(typeContractID string, name string, version int) reference.Global {
	contractID := fmt.Sprintf("%s::%s::v%02d", typeContractID, name, version)
	return GenerateClassReferenceFromCode(pulse.BuiltinContract, []byte(contractID))
}

// deprecated
// Generate code reference from contract id.
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
