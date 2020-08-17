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
	} else if err := validCallType(m.CallType); err != nil {
		return err
	}

	switch {
	case !m.GetCallFlags().IsValid():
		return throw.New("CallFlags should be valid")
	case m.ReturnArguments == nil:
		return throw.New("ReturnArguments should not be empty")
	}

	callerPulse, err := validSelfScopedGlobalWithPulseSpecialOrBeforeOrEq(m.Caller, currentPulse, "Caller")
	if err != nil {
		return err
	}

	calleePulse, err := validSelfScopedGlobalWithPulseSpecialOrBeforeOrEq(m.Callee, currentPulse, "Callee")
	if err != nil {
		return err
	}

	outgoingLocalPulse, err := validRequestGlobalWithPulseBeforeOrEq(m.CallOutgoing, currentPulse, "CallOutgoing")
	if err != nil {
		return err
	}

	incomingLocalPulse, err := validRequestGlobalWithPulseBeforeOrEq(m.CallIncoming, currentPulse, "CallIncoming")
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
