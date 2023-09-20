package pulsestor

import (
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

// GenesisPulse is a first pulse for the system
// DEPRECATED
var GenesisPulse = beat.Beat{ Data: pulse.Data{
	PulseNumber: pulse.MinTimePulse,
	DataExt : pulse.DataExt{
		PulseEpoch:  pulse.MinTimePulse,
		Timestamp: pulse.UnixTimeOfMinTimePulse,
	}}}

