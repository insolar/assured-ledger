// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VCallRequest{}

func validCallType(ct CallType) error {
	switch ct {
	case CTMethod, CTConstructor:
	case CTInboundAPICall, CTOutboundAPICall, CTNotifyCall, CTSAGACall, CTParallelCall, CTScheduleCall:
		return throw.New("CallType is not implemented")
	case CTInvalid:
		fallthrough
	default:
		return throw.New("CallType must be valid")
	}

	return nil
}

func (m *VCallRequest) validateUnimplemented() error {
	switch {
	case !m.CallSiteDeclaration.IsZero():
		return throw.New("CallSiteDeclaration should be empty")
	case !m.CallReason.IsZero():
		return throw.New("CallReason should be empty")
	case !m.RootTX.IsZero():
		return throw.New("RootTX should be empty")
	case !m.CallTX.IsZero():
		return throw.New("CallTX should be empty")
	case !m.ExpenseCenter.IsZero():
		return throw.New("ExpenseCenter should be empty")
	case !m.ResourceCenter.IsZero():
		return throw.New("ResourceCenter should be empty")
	case m.PayloadHash != nil:
		return throw.New("PayloadHash should be nil")
	case m.ProducerSignature != nil:
		return throw.New("ProducerSignature should be nil")
	case m.RegistrarSignature != nil:
		return throw.New("RegistrarSignature should be nil")
	case !m.RegistrarDelegationSpec.IsZero():
		return throw.New("RegistrarDelegationSpec should be empty")
	case !m.KnownCalleeIncoming.IsZero():
		return throw.New("KnownCalleeIncoming should be empty")
	case m.EntryHeadHash != nil:
		return throw.New("EntryHeadHash should be nil")
	case !m.TXExpiry.IsUnknown():
		return throw.New("TXExpiry should be unknown")
	case m.SecurityContext != nil:
		return throw.New("SecurityContext should be nil")
	case m.TXContext != nil:
		return throw.New("TXContext should be nil")
	case m.ExtensionHashes != nil:
		return throw.New("ExtensionHashes should be nil")
	case m.Extensions != nil:
		return throw.New("Extensions should be nil")
	case !m.CallAsOf.IsUnknown():
		return throw.New("CallAsOf should be zero")
	}

	return nil
}

func (m *VCallRequest) Validate(currentPulse PulseNumber) error {
	if err := validCallType(m.GetCallType()); err != nil {
		return err
	}

	switch {
	case !m.GetCallFlags().IsValid():
		return throw.New("CallFlags should be valid")
	case m.CallSiteMethod == "":
		return throw.New("CallSiteMethod shouldn't be empty")
	case !m.GetCallRequestFlags().IsValid():
		return throw.New("CallRequestFlags should be valid")
	case m.Arguments == nil:
		return throw.New("Arguments shouldn't be nil")
	}

	if !m.Caller.IsSelfScope() {
		return throw.New("Caller should be self scoped reference")
	}

	var (
		callerPulse = m.Caller.GetLocal().GetPulseNumber()
	)
	if !isSpecialOrTimePulseBeforeOrEq(callerPulse, currentPulse) {
		return throw.New("Caller should have special or valid time pulse before or equal to current pulse")
	}

	if !m.Callee.IsSelfScope() {
		return throw.New("Callee should be self scoped reference")
	}

	var (
		calleePulse = m.Caller.GetLocal().GetPulseNumber()
	)
	if !isSpecialOrTimePulseBeforeOrEq(calleePulse, currentPulse) {
		return throw.New("Callee should have special or valid time pulse before or equal to current pulse")
	}

	if m.CallOutgoing.IsEmpty() {
		return throw.New("CallOutgoing should be non-empty")
	}

	var (
		outgoingLocalPulse = m.CallOutgoing.GetLocal().GetPulseNumber()
		outgoingBasePulse  = m.CallOutgoing.GetBase().GetPulseNumber()
	)
	switch {
	case !isTimePulseBeforeOrEq(outgoingLocalPulse, currentPulse):
		return throw.New("CallOutgoing local part should have valid time pulse lesser or equal to current pulse")
	case !isSpecialOrTimePulseBeforeOrEq(outgoingBasePulse, currentPulse):
		// probably call outgoing base part can be special (API Call)
		return throw.New("CallOutgoing base part should have valid pulse lesser or equal to current pulse")
	case !globalBasePulseIsSpecialOrBeforeOrEqLocalPulse(m.CallOutgoing):
		return throw.New("CallOutgoing base pulse should be less or equal than local pulse")
	}

	switch {
	case !calleePulse.IsSpecial() && !calleePulse.IsBeforeOrEq(outgoingLocalPulse):
		return throw.New("Callee local pulse should be before or equal outgoing local pulse")
	case !callerPulse.IsSpecial() && !callerPulse.IsBeforeOrEq(outgoingLocalPulse):
		return throw.New("Caller local pulse should be before or equal outgoing local pulse")
	}

	return nil
}
