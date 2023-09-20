package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VFindCallRequest{}

func (m *VFindCallRequest) Validate(currentPulse PulseNumber) error {
	lookAtPulse := m.GetLookAt()
	if !isTimePulseBeforeOrEq(lookAtPulse, currentPulse) {
		return throw.New("LookAt should be valid time pulse before current pulse")
	}

	calleePulse, err := validSelfScopedGlobalWithPulseSpecialOrBefore(m.Callee.GetValue(), currentPulse, "Callee")
	if err != nil {
		return err
	}

	outgoingLocalPulse, err := validRequestGlobalWithPulseBeforeOrEq(m.Outgoing.GetValue(), currentPulse, "CallOutgoing")
	if err != nil {
		return err
	}

	// lookAtPulse >= outgoing >= calleePulse,
	if !lookAtPulse.IsEqOrAfter(outgoingLocalPulse) {
		return throw.New("LookAt should be more or equal Outgoing local pulse")
	} else if !outgoingLocalPulse.IsEqOrAfter(calleePulse) {
		return throw.New("Outgoing local pulse should be more or equal Callee pulse")
	}

	return nil
}
