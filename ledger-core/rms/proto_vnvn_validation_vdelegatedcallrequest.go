package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VDelegatedCallRequest{}

func (m *VDelegatedCallRequest) validateUnimplemented() error {
	return nil
}

func (m *VDelegatedCallRequest) Validate(currentPulse pulse.Number) error {
	if err := m.validateUnimplemented(); err != nil {
		return err
	} else if !m.GetCallFlags().IsValid() {
		return throw.New("CallFlags should be valid")
	}

	calleePulse, err := validSelfScopedGlobalWithPulseSpecialOrBefore(m.Callee.GetValue(), currentPulse, "Callee")
	if err != nil {
		return err
	}

	outgoingLocalPulse, err := validRequestGlobalWithPulseBefore(m.CallOutgoing.GetValue(), currentPulse, "CallOutgoing")
	if err != nil {
		return err
	}

	incomingLocalPulse, err := validRequestGlobalWithPulseBefore(m.CallIncoming.GetValue(), currentPulse, "CallIncoming")
	if err != nil {
		return err
	}

	switch {
	case !outgoingLocalPulse.IsEqOrAfter(incomingLocalPulse):
		return throw.New("CallOutgoing pulse should be more or equal than CallIncoming pulse")
	case !incomingLocalPulse.IsEqOrAfter(calleePulse):
		return throw.New("Callee pulse should be less or equal than CallOutgoing pulse")
	}

	return nil
}
