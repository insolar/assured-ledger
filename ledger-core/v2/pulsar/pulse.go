// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsar

import (
	pulse2 "github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulsar/entropygenerator"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

// NewPulse creates a new pulse with using of custom GeneratedEntropy Generator
func NewPulse(numberDelta uint32, previousPulseNumber pulse.Number, entropyGenerator entropygenerator.EntropyGenerator) *pulse2.Pulse {
	previousPulseNumber += pulse.Number(numberDelta)
	return &pulse2.Pulse{
		PulseNumber:     previousPulseNumber,
		NextPulseNumber: previousPulseNumber + pulse.Number(numberDelta),
		Entropy:         entropyGenerator.GenerateEntropy(),
	}
}
