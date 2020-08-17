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
	} else if !m.GetCallFlags().IsValid() {
		return throw.New("CallFlags should be valid")
	}

	calleePulse, err := validSelfScopedGlobalWithPulseSpecialOrBefore(m.Callee, currentPulse, "Callee")
	if err != nil {
		return err
	}

	outgoingLocalPulse, err := validOutgoingWithPulseBefore(m.CallOutgoing, currentPulse, "CallOutgoing")
	if err != nil {
		return err
	}

	incomingLocalPulse, err := validOutgoingWithPulseBefore(m.CallIncoming, currentPulse, "CallIncoming")
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
