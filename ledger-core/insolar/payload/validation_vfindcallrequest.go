// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VFindCallRequest{}

func (m *VFindCallRequest) Validate(currPulse PulseNumber) error {
	var (
		lookAtPulse   = m.GetLookAt()
		calleePulse   = m.GetCallee().GetLocal().Pulse()
		outgoingPulse = m.GetOutgoing().GetLocal().Pulse()
	)

	if !m.Callee.IsSelfScope() {
		return throw.New("callee should is self scope")
	}

	switch {
	case !lookAtPulse.IsTimePulse():
		return throw.New("invalid lookAt pulse")
	case !calleePulse.IsTimePulse():
		return throw.New("invalid callee pulse")
	case !outgoingPulse.IsTimePulse():
		return throw.New("invalid outgoing pulse")
	}

	if calleePulse > lookAtPulse {
		return throw.New("lookAt pulse should be more or equals callee pulse")
	}

	if calleePulse > outgoingPulse {
		return throw.New("outgoing pulse should be more or equals callee pulse")
	}

	if outgoingPulse > lookAtPulse {
		return throw.New("lookAt pulse should be more or equals outgoing pulse")
	}

	switch {
	case lookAtPulse > currPulse:
		return throw.New("lookAt pulse should be less or equals current pulse")
	case calleePulse > currPulse:
		return throw.New("callee pulse should be less or equals current pulse")
	case outgoingPulse > currPulse:
		return throw.New("outgoing pulse should be less or equals current pulse")
	}

	return nil
}
