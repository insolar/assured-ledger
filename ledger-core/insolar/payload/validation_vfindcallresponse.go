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
	if lookedAt := m.GetLookedAt(); !lookedAt.IsTimePulse() || lookedAt >= currentPulse {
		return throw.New("LookedAt should be valid pulse lesser than current pulse")
	}

	if callee := m.GetCallee(); !callee.IsSelfScope() {
		return throw.New("Callee should be self scoped reference")
	} else if pn := m.GetCallee().GetLocal().Pulse(); !pn.IsTimePulse() || pn >= currentPulse {
		return throw.New("Callee pulse should be valid pulse lesser than current pulse")
	}

	if m.GetOutgoing().IsEmpty() {
		return throw.New("Outgoing local should be non-empty ")
	} else if pn := m.GetOutgoing().GetLocal().Pulse(); !pn.IsTimePulse() || pn >= currentPulse {
		return throw.New("Outgoing local pulse should be valid time pulse lesser than current")
	} else if !pn.IsSpecialOrTimePulse() || pn >= currentPulse {
		// call outgoing base part can be special (API Call)
		return throw.New("Outgoing base pulse should be valid pulse lesser than current")
	} else if pn < m.GetCallee().GetLocal().Pulse() {
		return throw.New("Callee pulse should be more or equal than Outgoing pulse")
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
