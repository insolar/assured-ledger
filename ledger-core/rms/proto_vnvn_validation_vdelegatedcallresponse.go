// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VDelegatedCallResponse{}

func (m *VDelegatedCallResponse) Validate(currentPulse pulse.Number) error {
	calleePulse, err := validSelfScopedGlobalWithPulseSpecialOrBefore(m.Callee.GetGlobal(), currentPulse, "Callee")
	if err != nil {
		return err
	}

	incomingLocalPulse, err := validRequestGlobalWithPulseBefore(m.CallIncoming.GetGlobal(), currentPulse, "CallIncoming")
	if err != nil {
		return err
	}

	switch {
	case !incomingLocalPulse.IsEqOrAfter(calleePulse):
		return throw.New("Callee pulse should be less or equal than CallIncoming pulse")
	}

	return nil
}
