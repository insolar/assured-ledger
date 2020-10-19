// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

func (m *VObjectValidationReport) Validate(currentPulse PulseNumber) error {
	object := m.Object.GetValue()
	objectPulse, err := validSelfScopedGlobalWithPulseBeforeOrEq(object, currentPulse, "Object")
	if err != nil {
		return err
	}

	validated := m.Validated.GetValue()
	switch {
	case !isTimePulseBeforeOrEq(m.In, currentPulse):
		return throw.New("In should be valid time pulse before or equal current pulse")
	case !objectPulse.IsBefore(m.In):
		return throw.New("Object pulse should be before In pulse")
	case !validated.GetBase().Equal(object.GetBase()):
		return throw.New("Validated should be record of Object")
	}

	return nil
}
