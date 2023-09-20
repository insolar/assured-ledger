package beat

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

type Beat struct {
	BeatSeq uint32
	pulse.Data
	Range       pulse.Range
	StartedAt   time.Time
	Online      census.OnlinePopulation
	PulseOrigin []byte
}

func (v Beat) IsZero() bool {
	return v.Data.IsEmpty()
}
