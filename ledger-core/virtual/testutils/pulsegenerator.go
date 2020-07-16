// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testutils

import (
	"crypto/rand"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/appctl"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/census"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type PulseGenerator struct {
	prev      appctl.PulseChange
	last      appctl.PulseChange
	seqCount  uint32
	delta     uint16
}

func NewPulseGenerator(delta uint16, online census.OnlinePopulation) *PulseGenerator {
	return &PulseGenerator{
		delta: delta,
		last: appctl.PulseChange{
			Online: online,
		},
	}
}

func (g *PulseGenerator) GetLastPulseData() pulse.Data {
	return g.GetLastPulseAsPulse().Data
}

func (g *PulseGenerator) GetLastPulseAsPulse() appctl.PulseChange {
	if g.seqCount == 0 {
		panic(throw.IllegalState())
	}
	return g.last
}

func (g *PulseGenerator) GetPrevPulseAsPulse() appctl.PulseChange {
	if g.seqCount <= 1 {
		panic(throw.IllegalState())
	}
	return g.prev
}

func generateEntropy() (entropy longbits.Bits256) {
	if _, err := rand.Read(entropy[:]); err != nil {
		panic(err)
	}
	return
}

func (g *PulseGenerator) Generate() pulse.Data {
	if g.seqCount == 0 {
		g.last.Data = pulse.NewFirstPulsarData(g.delta, generateEntropy())
	} else {
		g.prev = g.last
		g.last = appctl.PulseChange{
			Data: g.last.CreateNextPulsarPulse(g.delta, generateEntropy),
			Online: g.prev.Online,
		}
	}
	g.seqCount++
	g.last.PulseSeq = g.seqCount
	g.last.StartedAt = time.Now()

	return g.last.Data
}
