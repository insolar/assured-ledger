// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

func (m *VCachedMemoryRequest) Validate(currentPulse PulseNumber) error {
	object := m.Object.GetValue()
	_, err := validSelfScopedGlobalWithPulseBeforeOrEq(object, currentPulse, "Object")
	if err != nil {
		return err
	}

	if m.Object.IsEmpty() {
		return throw.New("Object should not be empty")
	}

	if m.StateID.IsEmpty() {
		return throw.New("StateID should not be empty")
	}

	return nil
}
