// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsestor

import (
	"bytes"
	"encoding/json"

	"github.com/insolar/assured-ledger/ledger-core/appctl"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
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
// DEPRECATED
var GenesisPulse = appctl.PulseChange{ Data: pulse.Data{
	PulseNumber: pulse.MinTimePulse,
	DataExt : pulse.DataExt{
		PulseEpoch:  pulse.MinTimePulse,
		Timestamp: pulse.UnixTimeOfMinTimePulse,
	}}}

// Test use only
// DEPRECATED
var EphemeralPulse = appctl.PulseChange{ Data: pulse.Data{
	PulseNumber: pulse.MinTimePulse,
	DataExt : pulse.DataExt{
		PulseEpoch:  pulse.EphemeralPulseEpoch,
		Timestamp: pulse.UnixTimeOfMinTimePulse,
	}}}
