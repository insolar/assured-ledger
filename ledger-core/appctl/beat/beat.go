// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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
