// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package tests

import (
	"math/rand"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
)

func randBits256() longbits.Bits256 {
	v := longbits.Bits256{}
	_, _ = rand.Read(v[:])
	return v
}

func CreateGenerator(pulseCount int, pulseDelta uint16, output chan<- interface{}) {
	var pulseNum pulse.Number = 100000
	for i := 0; i < pulseCount; i++ {
		prevDelta := pulseDelta
		if i == 0 {
			prevDelta = 0
		}
		output <- WrapPacketParser(&EmuPulsarNetPacket{
			pulseData: pulse.NewPulsarData(pulseNum, pulseDelta, prevDelta, randBits256()),
		})

		pulseNum += pulse.Number(pulseDelta)
		time.Sleep(time.Duration(pulseDelta) * time.Second)
	}
}
