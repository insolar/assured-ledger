package entropygenerator

import (
	"crypto/rand"

	"github.com/insolar/assured-ledger/ledger-core/rms"
)

// EntropyGenerator is the base interface for generation of entropy for pulses
type EntropyGenerator interface {
	GenerateEntropy() rms.Entropy
}

// StandardEntropyGenerator is the base impl of EntropyGenerator with using of crypto/rand
type StandardEntropyGenerator struct {
}

// GenerateEntropy generate entropy with using of EntropyGenerator
func (generator *StandardEntropyGenerator) GenerateEntropy() rms.Entropy {
	entropy := make([]byte, rms.EntropySize)
	_, err := rand.Read(entropy)
	if err != nil {
		panic(err)
	}
	var result rms.Entropy
	copy(result[:], entropy[:rms.EntropySize])
	return result
}
