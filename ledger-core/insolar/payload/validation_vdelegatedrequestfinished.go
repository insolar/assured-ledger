// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VDelegatedRequestFinished{}

func (m *VDelegatedRequestFinished) validateUnimplemented() error {
	switch {
	case m.ResultFlags != nil:
		return throw.New("ResultFlags should be empty")
	case m.EntryHeadHash != nil:
		return throw.New("EntryHeadHash should be empty")
	}
	return nil
}

func (m *VDelegatedRequestFinished) isIntolerable() bool {
	return m.GetCallFlags().GetInterference() == contract.CallIntolerable
}

func (m *VDelegatedRequestFinished) Validate(currentPulse pulse.Number) error {
	if err := m.validateUnimplemented(); err != nil {
		return err
	}

	if err := validCallType(m.GetCallType()); err != nil {
		return err
	}

	if !m.GetCallFlags().IsValid() {
		return throw.New("CallFlags should be valid")
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
	case !isSpecialOrTimePulseBefore(incomingBasePulse, currentPulse):
		return throw.New("CallIncoming base part should have valid time pulse lesser than current pulse")
	case !globalBasePulseIsSpecialOrBeforeOrEqLocalPulse(m.CallIncoming):
		return throw.New("CallIncoming base pulse should be less or equal than local pulse")
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

	switch {
	case !outgoingLocalPulse.IsEqOrAfter(incomingLocalPulse):
		return throw.New("CallOutgoing pulse should be more or equal than CallIncoming pulse")
	case !incomingLocalPulse.IsEqOrAfter(calleePulse):
		return throw.New("Callee pulse should be less or equal than CallOutgoing pulse")
	case m.isIntolerable() && m.LatestState != nil:
		return throw.New("LatestState should be empty on Intolerable call")
	case m.CallType == CTConstructor && m.LatestState == nil:
		return throw.New("LatestState should be non-empty on Constructor call")
	}

	return nil
}
