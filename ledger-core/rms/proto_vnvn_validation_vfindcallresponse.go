package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VFindCallResponse{}

func (m *VFindCallResponse) Validate(currentPulse PulseNumber) error {
	if !isTimePulseBefore(m.LookedAt, currentPulse) {
		return throw.New("LookedAt should be valid pulse lesser than current pulse")
	}

	calleePulse, err := validSelfScopedGlobalWithPulseSpecialOrBefore(m.Callee.GetValue(), currentPulse, "Callee")
	if err != nil {
		return err
	}

	outgoingLocalPulse, err := validRequestGlobalWithPulseBefore(m.Outgoing.GetValue(), currentPulse, "Outgoing")
	if err != nil {
		return err
	}

	if !outgoingLocalPulse.IsEqOrAfter(calleePulse) {
		return throw.New("Callee pulse should be before outgoing pulse")
	}

	switch m.GetStatus() {
	case CallStateMissing:
		fallthrough
	case CallStateUnknown:
		if m.CallResult != nil {
			return throw.New("Call result should be empty")
		}
	case CallStateFound:
		if m.CallResult != nil {
			return m.CallResult.Validate(currentPulse)
		}
	}

	return nil
}
