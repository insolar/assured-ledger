// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VCallResult{}

func (m *VCallResult) validateUnimplemented() error {
	switch {
	case m.Extensions != nil:
		return throw.New("Extensions should be empty")
	case m.ExtensionHashes != nil:
		return throw.New("ExtensionHashes should be empty")
	case m.SecurityContext != nil:
		return throw.New("SecurityContext should be empty")
	case m.EntryHeadHash != nil:
		return throw.New("EntryHeadHash should be empty")
	case !m.RegistrarDelegationSpec.IsZero():
		return throw.New("RegistrarDelegationSpec should be zero")
	case m.RegistrarSignature != nil:
		return throw.New("RegistrarSignature should be empty")
	case m.ProducerSignature != nil:
		return throw.New("ProducerSignature should be empty")
	case m.PayloadHash != nil:
		return throw.New("PayloadHash should be empty")
	case m.ResultFlags != nil:
		return throw.New("ResultFlags should be empty")
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
	}

	switch m.GetCallType() {
	case CTMethod, CTConstructor:
	case CTInboundAPICall, CTOutboundAPICall, CTNotifyCall, CTSAGACall, CTParallelCall, CTScheduleCall:
		return throw.New("CallType is not implemented")
	case CTInvalid:
		fallthrough
	default:
		return throw.New("CallType must be valid")
	}

	switch {
	case !m.GetCallFlags().IsValid():
		return throw.New("CallFlags should be valid")
	case m.ReturnArguments == nil:
		return throw.New("ReturnArguments should not be empty")
	}

	var (
		calleePulse = m.Callee.GetLocal().GetPulseNumber()
		callerPulse = m.Caller.GetLocal().GetPulseNumber()
	)
	{ // Caller & Callee

		switch {
		case !m.Caller.IsSelfScope():
			return throw.New("Caller must be valid self scoped Reference")
		case !m.Callee.IsSelfScope():
			return throw.New("Callee must be valid self scoped Reference")
		case !isSpecialTimePulseBeforeOrEq(calleePulse, currentPulse):
			return throw.New("Callee must have valid time pulse with pulse lesser than current pulse")
		case !isSpecialTimePulseBeforeOrEq(callerPulse, currentPulse):
			return throw.New("Caller should have special or valid time pulse before or equal to current pulse")
		}
	}

	{ // CallOutgoing & CallIncoming
		switch {
		case m.CallOutgoing.IsEmpty():
			return throw.New("CallOutgoing should be non-empty")
		case m.CallIncoming.IsEmpty():
			return throw.New("CallIncoming should be non-empty")
		case !globalBasePulseBeforeOrEqLocalPulse(m.CallOutgoing):
			return throw.New("CallOutgoing base pulse should be less or equal than local pulse")
		case !globalBasePulseBeforeOrEqLocalPulse(m.CallIncoming):
			return throw.New("CallIncoming base pulse should be less or equal than local pulse")
		}

		var (
			outgoingLocalPulse = m.CallOutgoing.GetLocal().GetPulseNumber()
			outgoingBasePulse  = m.CallOutgoing.GetBase().GetPulseNumber()
		)
		switch {
		case !isTimePulseBeforeOrEq(outgoingLocalPulse, currentPulse):
			return throw.New("CallOutgoing local part should have valid time pulse lesser than current pulse")
		case !isSpecialTimePulseBeforeOrEq(outgoingBasePulse, currentPulse):
			// probably call outgoing base part can be special (API Call)
			return throw.New("CallOutgoing base part should have valid pulse lesser than current pulse")
		case !globalBasePulseBeforeOrEqLocalPulse(m.CallOutgoing):
			return throw.New("CallOutgoing base pulse should be less or equal than local pulse")
		}

		switch {
		case !calleePulse.IsSpecial() && !calleePulse.IsBeforeOrEq(outgoingLocalPulse):
			return throw.New("Callee local pulse should be before or equal outgoing local pulse")
		case !callerPulse.IsSpecial() && !callerPulse.IsBeforeOrEq(outgoingLocalPulse):
			return throw.New("Caller local pulse should be before or equal outgoing local pulse")
		}

	}

	return nil
}
