// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulse

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

func FromProto(p *PulseProto) *Pulse {
	result := &Pulse{
		PulseNumber:      p.PulseNumber,
		PrevPulseNumber:  p.PrevPulseNumber,
		NextPulseNumber:  p.NextPulseNumber,
		PulseTimestamp:   p.PulseTimestamp,
		EpochPulseNumber: pulse.Epoch(p.EpochPulseNumber),
		Entropy:          p.Entropy,
		Signs:            map[string]SenderConfirmation{},
	}
	copy(result.OriginID[:], p.OriginID)
	for _, sign := range p.Signs {
		pk, confirmation := SenderConfirmationFromProto(sign)
		result.Signs[pk] = confirmation
	}
	return result
}

func ToProto(p *Pulse) *PulseProto {
	result := &PulseProto{
		PulseNumber:      p.PulseNumber,
		PrevPulseNumber:  p.PrevPulseNumber,
		NextPulseNumber:  p.NextPulseNumber,
		PulseTimestamp:   p.PulseTimestamp,
		EpochPulseNumber: int32(p.EpochPulseNumber),
		OriginID:         p.OriginID[:],
		Entropy:          p.Entropy,
	}
	for pk, sign := range p.Signs {
		result.Signs = append(result.Signs, SenderConfirmationToProto(pk, sign))
	}
	return result
}

func SenderConfirmationToProto(publicKey string, p SenderConfirmation) *PulseSenderConfirmationProto {
	return &PulseSenderConfirmationProto{
		PublicKey:       publicKey,
		PulseNumber:     p.PulseNumber,
		ChosenPublicKey: p.ChosenPublicKey,
		Entropy:         p.Entropy,
		Signature:       p.Signature,
	}
}

func SenderConfirmationFromProto(p *PulseSenderConfirmationProto) (string, SenderConfirmation) {
	return p.PublicKey, SenderConfirmation{
		PulseNumber:     p.PulseNumber,
		ChosenPublicKey: p.ChosenPublicKey,
		Entropy:         p.Entropy,
		Signature:       p.Signature,
	}
}
