// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VDelegatedCallRequest{}

func (m *VDelegatedCallRequest) validateUnimplemented() error {
	if !m.RecordHead.IsEmpty() {
		return throw.New("RecordHead should be empty")
	}
	return nil
}

func (m *VDelegatedCallRequest) Validate(currentPulse pulse.Number) error {
	if err := m.validateUnimplemented(); err != nil {
		return err
	}

	if !m.Callee.IsSelfScope() {
		return throw.New("Callee must be valid self scoped Reference")
	}

	var (
		calleePulse = m.Callee.GetLocal().GetPulseNumber()
	)
	if !calleePulse.IsTimePulse() || !calleePulse.IsBefore(currentPulse) {
		return throw.New("Callee must have valid time pulse with pulse lesser than current pulse")
	}

	if !m.GetCallFlags().IsValid() {
		return throw.New("CallFlags should be valid")
	}

	if m.CallOutgoing.IsEmpty() {
		return throw.New("CallOutgoing should be non-empty")
	}

	var (
		outgoingLocalPulse = m.CallOutgoing.GetLocal().GetPulseNumber()
		outgoingBasePulse  = m.CallOutgoing.GetBase().GetPulseNumber()
	)
	switch {
	case !isTimePulseBefore(outgoingLocalPulse, currentPulse):
		return throw.New("CallOutgoing local part should have valid time pulse lesser than current pulse")
	case !isSpecialOrTimePulseBefore(outgoingBasePulse, currentPulse):
		// probably call outgoing base part can be special (API Call)
		return throw.New("CallOutgoing base part should have valid pulse lesser than current pulse")
	case !globalBasePulseIsSpecialOrBeforeOrEqLocalPulse(m.CallOutgoing):
		return throw.New("CallOutgoing base pulse should be less or equal than local pulse")
	}

	if m.CallIncoming.IsEmpty() {
		return throw.New("CallIncoming should be non-empty")
	}

	var (
		incomingLocalPulse = m.CallIncoming.GetLocal().GetPulseNumber()
		incomingBasePulse  = m.CallIncoming.GetBase().GetPulseNumber()
	)
	switch {
	case !isTimePulseBefore(incomingLocalPulse, currentPulse):
		return throw.New("CallIncoming local part should have valid time pulse lesser than current pulse")
	case !isTimePulseBefore(incomingBasePulse, currentPulse):
		return throw.New("CallIncoming base part should have valid time pulse lesser than current pulse")
	case !globalBasePulseIsSpecialOrBeforeOrEqLocalPulse(m.CallIncoming):
		return throw.New("CallIncoming base pulse should be less or equal than local pulse")
	}

	switch {
	case !outgoingLocalPulse.IsEqOrAfter(incomingLocalPulse):
		return throw.New("CallOutgoing pulse should be more or equal than CallIncoming pulse")
	case !incomingLocalPulse.IsEqOrAfter(calleePulse):
		return throw.New("Callee pulse should be less or equal than CallOutgoing pulse")
	}

	return nil
}
