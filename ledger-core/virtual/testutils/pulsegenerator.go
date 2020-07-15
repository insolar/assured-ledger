// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testutils

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/appctl"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type PulseGenerator struct {
	delta     uint16
	pulseList []appctl.PulseChange
}

func NewPulseGenerator(delta uint16) *PulseGenerator {
	g := &PulseGenerator{delta: delta}
	g.appendPulse(pulse.Data{
		PulseNumber: pulse.MinTimePulse,
		DataExt:     pulse.DataExt{
			PulseEpoch:     pulse.MinTimePulse,
			NextPulseDelta: delta,
			PrevPulseDelta: 0,
		},
	})
	return g
}

func (g PulseGenerator) GetLastPulseData() pulse.Data {
	return g.pulseList[len(g.pulseList)-1].Data
}

func (g PulseGenerator) GetLastPulseAsPulse() appctl.PulseChange {
	return g.pulseList[len(g.pulseList)-1]
}

func (g PulseGenerator) GetPrevPulseAsPulse() appctl.PulseChange {
	if len(g.pulseList) < 2 {
		panic(throw.IllegalValue())
	}
	return g.pulseList[len(g.pulseList)-2]
}

func (g *PulseGenerator) appendPulse(data pulse.Data) {
	g.pulseList = append(g.pulseList, appctl.PulseChange{
		PulseSeq:    uint32(len(g.pulseList) + 1),
		Data:        data,
		Pulse:       data.AsRange(),
		StartedAt:   time.Now(),
		Census:      nil, // TODO census
	})
}

func generateEntropy() longbits.Bits256 {
	var entropy longbits.Bits256

	actualLength, err := rand.Read(entropy[:])
	if err != nil {
		panic(fmt.Sprintf("failed to read %d random bytes: %s", len(entropy), err.Error()))
	} else if actualLength != len(entropy) {
		panic(fmt.Sprintf("unreachable, %d != %d", len(entropy), actualLength))
	}

	return entropy
}

func (g PulseGenerator) generateNextPulse() pulse.Data {
	var (
		nextPulse pulse.Data
		prevPulse = g.GetLastPulseData()
	)

	if !prevPulse.IsEmpty() || prevPulse.PulseNumber == pulse.MinTimePulse {
		nextPulse = prevPulse.CreateNextPulsarPulse(g.delta, generateEntropy)
	} else {
		nextPulse = pulse.NewFirstPulsarData(g.delta, generateEntropy())
	}

	return nextPulse
}

func (g *PulseGenerator) Generate() pulse.Data {
	newPulseData := g.generateNextPulse()
	g.appendPulse(newPulseData)
	return newPulseData
}
