// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package launchnet

import (
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/pulsar"
	"github.com/insolar/assured-ledger/ledger-core/pulsar/entropygenerator"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type PulseGenerator struct {
	history []pulsar.PulsePacket
	delta   uint16
	lock    sync.Mutex
}

func NewPulseGenerator(delta uint16) *PulseGenerator {
	return &PulseGenerator{
		delta: delta,
	}
}

func (pg *PulseGenerator) Last() (pulsar.PulsePacket, error) {
	pg.lock.Lock()
	defer pg.lock.Unlock()

	if len(pg.history) == 0 {
		return pulsar.PulsePacket{}, throw.New("history is empty")
	}

	return pg.history[len(pg.history)-1], nil
}

func (pg *PulseGenerator) CountBack(count uint) (pulsar.PulsePacket, error) {
	pg.lock.Lock()
	defer pg.lock.Unlock()
	if int(count) >= len(pg.history) {
		return pulsar.PulsePacket{}, throw.New("count is too big")
	}

	return pg.history[len(pg.history)-int(count)-1], nil
}

func (pg *PulseGenerator) Generate() pulsar.PulsePacket {
	pg.lock.Lock()
	defer pg.lock.Unlock()

	var lastPN pulse.Number
	if len(pg.history) == 0 {
		lastPN = pulse.OfNow()
	} else {
		lastPN = pg.history[len(pg.history)-1].PulseNumber
	}
	pulsePacket := makePacket(lastPN, pg.delta)
	pg.history = append(pg.history, pulsePacket)
	return pulsePacket
}

func makePacket(prevPN pulse.Number, delta uint16) pulsar.PulsePacket {
	entropyGen := entropygenerator.StandardEntropyGenerator{}
	lastPN := prevPN + pulse.Number(delta)
	return pulsar.PulsePacket{
		PulseNumber:      lastPN,
		Entropy:          entropyGen.GenerateEntropy(),
		NextPulseNumber:  lastPN + pulse.Number(delta),
		PrevPulseNumber:  prevPN,
		EpochPulseNumber: lastPN.AsEpoch(),
		PulseTimestamp:   time.Now().UnixNano(),
		Signs:            map[string]pulsar.SenderConfirmation{},
	}
}
