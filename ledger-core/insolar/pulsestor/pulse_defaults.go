// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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

