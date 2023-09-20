package rms

import (
	"bytes"
	"encoding/json"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const (
	// EntropySize declares the number of bytes in the pulse entropy
	EntropySize = 32
	// OriginIDSize declares the number of bytes in the origin id
	OriginIDSize = 16
)

// Entropy is 64 random bytes used in every pseudo-random calculations.
type Entropy [EntropySize]byte

func (entropy Entropy) Marshal() ([]byte, error) { return entropy[:], nil }
func (entropy Entropy) MarshalTo(data []byte) (int, error) {
	copy(data, entropy[:])
	return EntropySize, nil
}
func (entropy *Entropy) Unmarshal(data []byte) error {
	if len(data) != EntropySize {
		return throw.New("Not enough bytes to unpack Entropy")
	}
	copy(entropy[:], data)
	return nil
}
func (entropy *Entropy) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, entropy)
}
func (entropy Entropy) ProtoSize() int { return EntropySize }
func (entropy Entropy) Compare(other Entropy) int {
	return bytes.Compare(entropy[:], other[:])
}
func (entropy Entropy) Equal(other Entropy) bool {
	return entropy == other
}

