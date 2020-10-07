// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func (m *VCachedMemoryResponse) Validate(currentPulse PulseNumber) error {
	object := m.Object.GetValue()
	_, err := validSelfScopedGlobalWithPulseBeforeOrEq(object, currentPulse, "Object")
	if err != nil {
		return err
	}

	switch m.CallStatus {
	case CachedMemoryStateFound:
		if m.Memory.IsEmpty() {
			return throw.New("Memory should not be empty")
		}
		fallthrough
	case CachedMemoryStateUnknown:
		// TODO need to add check for m.Class, it doesn't exist at master right now
		// TODO add check for correlation between object and class

		// TODO stateID and object are from one chain
		if m.StateID.IsEmpty() {
			return throw.New("StateID should not be empty")
		}

	default:
		return throw.New("unexpected CallStatus")
	}

	return nil
}
