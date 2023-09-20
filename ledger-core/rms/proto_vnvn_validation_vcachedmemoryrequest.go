package rms

import "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

func (m *VCachedMemoryRequest) Validate(currentPulse PulseNumber) error {
	object := m.Object.GetValue()
	_, err := validSelfScopedGlobalWithPulseBeforeOrEq(object, currentPulse, "Object")
	if err != nil {
		return err
	}

	// TODO stateID and object are from one chain
	if m.State.IsEmpty() {
		return throw.New("State should not be empty")
	}

	return nil
}
