// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VFindCallResponse{}

func (m *VFindCallResponse) Validate(currentPulse PulseNumber) error {
	if !isTimePulseBefore(m.LookedAt, currentPulse) {
		return throw.New("LookedAt should be valid pulse lesser than current pulse")
	}

	calleePulse, err := validSelfScopedGlobalWithPulseSpecialOrBefore(m.Callee, currentPulse, "Callee")
	if err != nil {
		return err
	}

	outgoingLocalPulse, err := validOutgoingWithPulseBefore(m.Outgoing, currentPulse, "Outgoing")
	if err != nil {
		return err
	}

	if !outgoingLocalPulse.IsEqOrAfter(calleePulse) {
		return throw.New("Callee pulse should be before outgoing pulse")
	}

	switch m.GetStatus() {
	case MissingCall:
		fallthrough
	case UnknownCall:
		if m.CallResult != nil {
			return throw.New("Call result should be empty")
		}
	case FoundCall:
		if m.CallResult != nil {
			return m.CallResult.Validate(currentPulse)
		}
	}

	return nil
}
