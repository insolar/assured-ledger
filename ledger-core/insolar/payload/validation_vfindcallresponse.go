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
	lookedAt := m.GetLookedAt()
	if !lookedAt.IsTimePulse() || lookedAt >= currentPulse {
		return throw.New("LookedAt should be valid pulse lesser than current pulse")
	}

	callee := m.GetCallee()
	calleePulse := m.GetCallee().GetLocal().Pulse()
	if !callee.IsSelfScope() {
		return throw.New("Callee should be self scoped reference")
	} else if !calleePulse.IsTimePulse() || calleePulse >= currentPulse {
		return throw.New("Callee pulse should be valid pulse lesser than current pulse")
	}

	outgoing := m.GetOutgoing()
	outgoingPulse := m.GetOutgoing().GetLocal().Pulse()
	if outgoing.IsEmpty() {
		return throw.New("Outgoing should be non-empty")
	} else if !outgoingPulse.IsTimePulse() || outgoingPulse >= currentPulse {
		return throw.New("Outgoing pulse should be valid pulse lesser than current pulse")
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
