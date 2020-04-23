// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package genesisrefs

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
	"github.com/insolar/assured-ledger/ledger-core/v2/platformpolicy"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

const (
	PrototypeType    = "prototype"
	PrototypeSuffix  = "_proto"
	FundsDepositName = "genesis_deposit"
)

// Generate reference from hash code.
func GenerateProtoReferenceFromCode(pulse insolar.PulseNumber, code []byte) *insolar.Reference {
	hasher := platformpolicy.NewPlatformCryptographyScheme().ReferenceHasher()
	codeHash := hasher.Hash(code)
	id := insolar.NewID(pulse, codeHash)
	return insolar.NewReference(*id)
}

// Generate prototype reference from contract id.
func GenerateProtoReferenceFromContractID(typeContractID string, name string, version int) *insolar.Reference {
	contractID := fmt.Sprintf("%s::%s::v%02d", typeContractID, name, version)
	return GenerateProtoReferenceFromCode(pulse.BuiltinContract, []byte(contractID))
}

// Generate contract reference from contract id.
func GenerateCodeReferenceFromContractID(typeContractID string, name string, version int) *insolar.Reference {
	contractID := fmt.Sprintf("%s::%s::v%02d", typeContractID, name, version)
	hasher := platformpolicy.NewPlatformCryptographyScheme().ReferenceHasher()
	codeHash := hasher.Hash([]byte(contractID))
	id := insolar.NewID(pulse.BuiltinContract, codeHash)
	return insolar.NewRecordReference(*id)
}

// GenesisRef returns reference to any genesis records.
func GenesisRef(name string) insolar.Reference {
	pcs := platformpolicy.NewPlatformCryptographyScheme()
	req := record.IncomingRequest{
		CallType: record.CTGenesis,
		Method:   name,
	}
	virtRec := record.Wrap(&req)
	hash := record.HashVirtual(pcs.ReferenceHasher(), virtRec)
	id := insolar.NewID(pulse.MinTimePulse, hash)
	return *insolar.NewReference(*id)
}
