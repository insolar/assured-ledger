package pulsar

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

// PulsePacket is base data structure for a pulse.
type PulsePacket struct {
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
