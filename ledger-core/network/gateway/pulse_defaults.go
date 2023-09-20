package gateway

import (
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

var EphemeralPulse = beat.Beat{ Data: pulse.Data{
	PulseNumber: pulse.MinTimePulse,
	DataExt : pulse.DataExt{
		PulseEpoch:  pulse.EphemeralPulseEpoch,
		Timestamp: pulse.UnixTimeOfMinTimePulse,
	}}}
