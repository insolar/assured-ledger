// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cloud

import (
	"context"
	"sync"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/pulsar"
	"github.com/insolar/assured-ledger/ledger-core/pulsar/entropygenerator"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type ManualPulsar struct {
	controller            *Controller
	delta                 uint16
	lastPN                pulse.Number
	lock                  sync.Mutex
	customPulseChangeUsed bool
}

func NewHandyPulsar(pulseDistr *Controller, delta uint16) *ManualPulsar {
	return &ManualPulsar{
		controller: pulseDistr,
		delta:      delta,
	}
}

func (hp *ManualPulsar) IncrementPulseForNode(ctx context.Context, ref reference.Global) {
	hp.lock.Lock()
	defer hp.lock.Unlock()

	hp.customPulseChangeUsed = true

	lastPN, err := hp.controller.GetLastPulseNumberOnNode(ref)
	if err != nil {
		panic(throw.W(err, "can't get last pulse"))
	}

	pulseForSending := makePacket(lastPN, hp.delta)
	err = hp.controller.ChangePulseForNode(ctx, ref, pulseForSending)
	if err != nil {
		panic(throw.W(err, "can't change pulse"))
	}
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

func (hp *ManualPulsar) IncrementPulse() {
	hp.lock.Lock()
	defer hp.lock.Unlock()

	if hp.customPulseChangeUsed {
		panic("attempt to distribute pulse change to all nodes after custom change")
	}

	if hp.lastPN == 0 {
		hp.lastPN = pulse.OfNow()
	}

	pulseForSending := makePacket(hp.lastPN, hp.delta)
	hp.lastPN += pulse.Number(hp.delta)

	hp.controller.Distribute(context.Background(), pulseForSending)
}
