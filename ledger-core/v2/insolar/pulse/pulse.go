// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulse

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

const (
	// EntropySize declares the number of bytes in the pulse entropy
	EntropySize = 64
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
		return errors.New("Not enough bytes to unpack Entropy")
	}
	copy(entropy[:], data)
	return nil
}
func (entropy *Entropy) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, entropy)
}
func (entropy Entropy) Size() int { return EntropySize }
func (entropy Entropy) Compare(other Entropy) int {
	return bytes.Compare(entropy[:], other[:])
}
func (entropy Entropy) Equal(other Entropy) bool {
	return entropy.Compare(other) == 0
}

//go:generate protoc -I$GOPATH/src -I./ --gogoslick_out=./ pulse.proto
//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse.Manager -s _mock.go -g

// Manager provides Ledger's methods related to Pulse.
type Manager interface {
	// Set set's new pulse and closes current jet drop. If dry is true, nothing will be saved to storage.
	Set(ctx context.Context, pulse Pulse) error
}

// Pulse is base data structure for a pulse.
type Pulse struct {
	PulseNumber     pulse.Number
	PrevPulseNumber pulse.Number
	NextPulseNumber pulse.Number

	PulseTimestamp   int64
	EpochPulseNumber pulse.Epoch
	OriginID         [OriginIDSize]byte

	Entropy Entropy
	Signs   map[string]SenderConfirmation
}

// SenderConfirmation contains confirmations of the pulse from other pulsars
// Because the system is using BFT for consensus between pulsars, because of it
// All pulsar send to the chosen pulsar their confirmations
// Every node in the network can verify the signatures
type SenderConfirmation struct {
	PulseNumber     pulse.Number
	ChosenPublicKey string
	Entropy         Entropy
	Signature       []byte
}

// GenesisPulse is a first pulse for the system
// because first 2 bits of pulse number and first 65536 pulses a are used by system needs and pulse numbers are related to the seconds of Unix time
// for calculation pulse numbers we use the formula = unix.Now() - firstPulseDate + 65536
var GenesisPulse = &Pulse{
	PulseNumber:      pulse.MinTimePulse,
	Entropy:          [EntropySize]byte{},
	EpochPulseNumber: pulse.MinTimePulse,
	PulseTimestamp:   pulse.UnixTimeOfMinTimePulse,
}

// EphemeralPulse is used for discovery network bootstrap
var EphemeralPulse = &Pulse{
	PulseNumber:      pulse.MinTimePulse,
	Entropy:          [EntropySize]byte{},
	EpochPulseNumber: pulse.EphemeralPulseEpoch,
	PulseTimestamp:   pulse.UnixTimeOfMinTimePulse,
}
