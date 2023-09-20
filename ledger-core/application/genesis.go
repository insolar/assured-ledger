package application

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/reference"
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
