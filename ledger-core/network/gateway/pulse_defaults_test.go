// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

// EphemeralPulse is for test use only
var EphemeralPulse = beat.Beat{ Data: pulse.Data{
	PulseNumber: pulse.MinTimePulse,
	DataExt : pulse.DataExt{
		PulseEpoch:  pulse.EphemeralPulseEpoch,
		Timestamp: pulse.UnixTimeOfMinTimePulse,
	}}}
