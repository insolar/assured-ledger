// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulsestor

import (
	"github.com/insolar/assured-ledger/ledger-core/appctl"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

// Pulse is base data structure for a pulse.
type Pulse struct {
	PulseNumber     pulse.Number
	PrevPulseNumber pulse.Number
	NextPulseNumber pulse.Number

	PulseTimestamp   int64
	EpochPulseNumber pulse.Epoch
	OriginID         [rms.OriginIDSize]byte

	Entropy rms.Entropy
	Signs   map[string]SenderConfirmation
}

// SenderConfirmation contains confirmations of the pulse from other pulsars
// Because the system is using BFT for consensus between pulsars, because of it
// All pulsar send to the chosen pulsar their confirmations
// Every node in the network can verify the signatures
type SenderConfirmation struct {
	PulseNumber     pulse.Number
	ChosenPublicKey string
	Entropy         rms.Entropy
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

// EphemeralPulse is for test use only
// DEPRECATED
var EphemeralPulse = appctl.PulseChange{ Data: pulse.Data{
	PulseNumber: pulse.MinTimePulse,
	DataExt : pulse.DataExt{
		PulseEpoch:  pulse.EphemeralPulseEpoch,
		Timestamp: pulse.UnixTimeOfMinTimePulse,
	}}}
