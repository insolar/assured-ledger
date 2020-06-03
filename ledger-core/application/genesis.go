// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package application

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

type genesisBinary reference.LocalHash

// GenesisRecord is initial chain record.
var GenesisRecord = genesisBinary(reference.LocalHash{0xAC})

// ID returns genesis record id.
func (r genesisBinary) ID() reference.Local {
	return reference.NewRecordID(pulsestor.GenesisPulse.PulseNumber, reference.LocalHash(r))
}

// Ref returns genesis record reference.
func (r genesisBinary) Ref() reference.Global {
	return reference.NewSelf(r.ID())
}

// GenesisHeavyConfig carries data required for initial genesis on heavy node.
type GenesisHeavyConfig struct {
	// Skip is flag for skipping genesis on heavy node. Useful for some test cases.
	Skip bool
}
