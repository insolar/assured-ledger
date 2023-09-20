package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VCallResult{}

func (m *VCallResult) validateUnimplemented() error {
	switch {
	case !m.SecurityContext.IsEmpty():
		return throw.New("SecurityContext should be empty")
	case !m.RegistrarDelegationSpec.IsZero():
		return throw.New("RegistrarDelegationSpec should be zero")
	case !m.RegistrarSignature.IsEmpty():
		return throw.New("RegistrarSignature should be empty")
	case !m.ProducerSignature.IsEmpty():
		return throw.New("ProducerSignature should be empty")
	case m.ResultFlags != 0:
		return throw.New("ResultFlags should be zero")
	case !m.CallIncomingResult.IsEmpty():
		return throw.New("CallIncomingResult should be empty")
	case !m.CallAsOf.IsUnknown():
		return throw.New("CallAsOf should be zero")
	}

	return nil
}

func (m *VCallResult) Validate(currentPulse PulseNumber) error {
	if err := m.validateUnimplemented(); err != nil {
		return err
	} else if err := validCallType(m.CallType); err != nil {
		return err
	}

	switch {
	case !m.GetCallFlags().IsValid():
		return throw.New("CallFlags should be valid")
	case m.ReturnArguments.IsEmpty():
		return throw.New("ReturnArguments should not be empty")
	}

	callerPulse, err := validSelfScopedGlobalWithPulseSpecialOrBeforeOrEq(m.Caller.GetValue(), currentPulse, "Caller")
	if err != nil {
		return err
	}

	calleePulse, err := validSelfScopedGlobalWithPulseSpecialOrBeforeOrEq(m.Callee.GetValue(), currentPulse, "Callee")
	if err != nil {
		return err
	}

	outgoingLocalPulse, err := validRequestGlobalWithPulseBeforeOrEq(m.CallOutgoing.GetValue(), currentPulse, "CallOutgoing")
	if err != nil {
		return err
	}

	incomingLocalPulse, err := validRequestGlobalWithPulseBeforeOrEq(m.CallIncoming.GetValue(), currentPulse, "CallIncoming")
	if err != nil {
		return err
	}

	switch {
	case !isSpecialOrTimePulseBeforeOrEq(calleePulse, outgoingLocalPulse):
		return throw.New("Callee local pulse should be before or equal outgoing local pulse")
	case !isSpecialOrTimePulseBeforeOrEq(callerPulse, outgoingLocalPulse):
		return throw.New("Caller local pulse should be before or equal outgoing local pulse")
	case !isSpecialOrTimePulseBeforeOrEq(calleePulse, incomingLocalPulse):
		return throw.New("Callee local pulse should be before or equal incoming local pulse")
	case !isSpecialOrTimePulseBeforeOrEq(callerPulse, incomingLocalPulse):
		return throw.New("Caller local pulse should be before or equal incoming local pulse")
	}

	return nil
}
