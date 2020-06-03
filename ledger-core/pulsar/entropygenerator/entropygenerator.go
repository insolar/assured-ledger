// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package entropygenerator

import (
	"crypto/rand"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulsestor"
)

// EntropyGenerator is the base interface for generation of entropy for pulses
type EntropyGenerator interface {
	GenerateEntropy() pulsestor.Entropy
}

// StandardEntropyGenerator is the base impl of EntropyGenerator with using of crypto/rand
type StandardEntropyGenerator struct {
}

// GenerateEntropy generate entropy with using of EntropyGenerator
func (generator *StandardEntropyGenerator) GenerateEntropy() pulsestor.Entropy {
	entropy := make([]byte, pulsestor.EntropySize)
	_, err := rand.Read(entropy)
	if err != nil {
		panic(err)
	}
	var result pulsestor.Entropy
	copy(result[:], entropy[:pulsestor.EntropySize])
	return result
}
