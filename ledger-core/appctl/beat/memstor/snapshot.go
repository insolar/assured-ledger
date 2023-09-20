package memstor

import (
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type Snapshot struct {
	pulse      pulse.Number
	population census.OnlinePopulation
}

func (s *Snapshot) GetPulseNumber() pulse.Number {
	return s.pulse
}

// NewSnapshot create new snapshot for pulse.
func NewSnapshot(number pulse.Number, population census.OnlinePopulation) *Snapshot {
	return &Snapshot{
		pulse: number,
		population:   population,
	}
}
