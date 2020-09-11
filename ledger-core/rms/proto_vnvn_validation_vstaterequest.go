// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VStateRequest{}

func (m *VStateRequest) validateUnimplemented() error {
	switch {
	case !m.RequestedContentLimit.IsEmpty():
		return throw.New("RequestedContentLimit should be empty")
	case !m.SupportedExtensions.IsEmpty():
		return throw.New("SupportedExtensions should be empty")
	case !m.ProducerSignature.IsEmpty():
		return throw.New("ProducerSignature should be empty")
	case m.CallRequestFlags != 0:
		return throw.New("CallRequestFlags should be zero")
	}

	return nil
}

func (m *VStateRequest) Validate(currentPulse PulseNumber) error {
	if err := m.validateUnimplemented(); err != nil {
		return err
	}

	objectPulse, err := validSelfScopedGlobalWithPulseBeforeOrEq(m.Object.GetValue(), currentPulse, "Object")
	if err != nil {
		return err
	}

	switch {
	case !isTimePulseBefore(m.AsOf, currentPulse):
		return throw.New("AsOf should be valid time pulse before current pulse")
	case !objectPulse.IsBeforeOrEq(m.AsOf):
		return throw.New("Object pulse should be before or equal AsOf pulse")
	case !m.RequestedContent.IsValid():
		return throw.New("RequestedContent should be valid")
	}

	return nil
}
