// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsar

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/pulsar/entropygenerator"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

// NewPulse creates a new pulse with using of custom GeneratedEntropy Generator
func NewPulse(numberDelta uint32, previousPulseNumber pulse.Number, entropyGenerator entropygenerator.EntropyGenerator) *pulsestor.Pulse {
	previousPulseNumber += pulse.Number(numberDelta)
	return &pulsestor.Pulse{
		PulseNumber:     previousPulseNumber,
		NextPulseNumber: previousPulseNumber + pulse.Number(numberDelta),
		Entropy:         entropyGenerator.GenerateEntropy(),
	}
}
