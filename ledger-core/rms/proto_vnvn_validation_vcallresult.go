// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VCallResult{}

func (m *VCallResult) validateUnimplemented() error {
	switch {
	case !m.Extensions.IsEmpty():
		return throw.New("Extensions should be empty")
	case !m.ExtensionHashes.IsEmpty():
		return throw.New("ExtensionHashes should be empty")
	case !m.SecurityContext.IsEmpty():
		return throw.New("SecurityContext should be empty")
	case !m.EntryHeadHash.IsEmpty():
		return throw.New("EntryHeadHash should be empty")
	case !m.RegistrarDelegationSpec.IsZero():
		return throw.New("RegistrarDelegationSpec should be zero")
	case !m.RegistrarSignature.IsEmpty():
		return throw.New("RegistrarSignature should be empty")
	case !m.ProducerSignature.IsEmpty():
		return throw.New("ProducerSignature should be empty")
	case !m.PayloadHash.IsEmpty():
		return throw.New("PayloadHash should be empty")
	case !m.ResultFlags.IsEmpty():
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
	} else if err := validCallType(m.CallType); err != nil {
		return err
	}

	switch {
	case !m.GetCallFlags().IsValid():
		return throw.New("CallFlags should be valid")
	case m.ReturnArguments.IsEmpty():
		return throw.New("ReturnArguments should not be empty")
	}

	callerPulse, err := validSelfScopedGlobalWithPulseSpecialOrBeforeOrEq(m.Caller.GetGlobal(), currentPulse, "Caller")
	if err != nil {
		return err
	}

	calleePulse, err := validSelfScopedGlobalWithPulseSpecialOrBeforeOrEq(m.Callee.GetGlobal(), currentPulse, "Callee")
	if err != nil {
		return err
	}

	outgoingLocalPulse, err := validRequestGlobalWithPulseBeforeOrEq(m.CallOutgoing.GetGlobal(), currentPulse, "CallOutgoing")
	if err != nil {
		return err
	}

	incomingLocalPulse, err := validRequestGlobalWithPulseBeforeOrEq(m.CallIncoming.GetGlobal(), currentPulse, "CallIncoming")
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
