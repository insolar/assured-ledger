// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsar

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulsar/entropygenerator"
)

// NewPulse creates a new pulse with using of custom GeneratedEntropy Generator
func NewPulse(numberDelta uint32, previousPulseNumber insolar.PulseNumber, entropyGenerator entropygenerator.EntropyGenerator) *insolar.Pulse {
	previousPulseNumber += insolar.PulseNumber(numberDelta)
	return &insolar.Pulse{
		PulseNumber:     previousPulseNumber,
		NextPulseNumber: previousPulseNumber + insolar.PulseNumber(numberDelta),
		Entropy:         entropyGenerator.GenerateEntropy(),
	}
}
