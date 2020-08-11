// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VDelegatedCallResponse{}

func (m *VDelegatedCallResponse) Validate(currentPulse pulse.Number) error {
	if !m.Callee.IsSelfScope() {
		return throw.New("Callee must be valid self scoped Reference")
	}

	var (
		calleePulse = m.Callee.GetLocal().GetPulseNumber()
	)
	if !calleePulse.IsTimePulse() || !calleePulse.IsBefore(currentPulse) {
		return throw.New("Callee must have valid time pulse with pulse lesser than current pulse")
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
	case !incomingLocalPulse.IsEqOrAfter(calleePulse):
		return throw.New("Callee pulse should be less or equal than CallOutgoing pulse")
	}

	return nil
}
