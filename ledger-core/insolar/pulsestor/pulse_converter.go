// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsestor

import (
	"time"

	"github.com/insolar/assured-ledger/ledger-core/appctl"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

func FromProto(p *rms.PulseProto) appctl.PulseChange {
	result := appctl.PulseChange{}
	result.PulseNumber = p.PulseNumber
	result.PrevPulseDelta = uint16(p.PulseNumber - p.PrevPulseNumber) // INCORRECT
	result.NextPulseDelta = uint16(p.NextPulseNumber - p.PulseNumber) // INCORRECT
	result.Timestamp = uint32(p.PulseTimestamp / int64(time.Second))
	result.PulseEpoch = pulse.Epoch(p.EpochPulseNumber)
	copy(result.PulseEntropy[:], p.Entropy[:])
	result.PulseOrigin = append([]byte(nil), p.OriginID...)
	return result
}

func ToProto(p appctl.PulseChange) *rms.PulseProto {
	result := &rms.PulseProto{
		PulseNumber:      p.PulseNumber,
		PrevPulseNumber:  p.PulseNumber - pulse.Number(p.PrevPulseDelta), // INCORRECT
		NextPulseNumber:  p.PulseNumber + pulse.Number(p.NextPulseDelta), // INCORRECT
		PulseTimestamp:   int64(p.Timestamp) * int64(time.Second),
		EpochPulseNumber: int32(p.PulseEpoch),
		OriginID:         p.PulseOrigin,
		// Entropy:          p.Entropy,
	}
	copy(result.Entropy[:], p.PulseEntropy[:])
	return result
}
