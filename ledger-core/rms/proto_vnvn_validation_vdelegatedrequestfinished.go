// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VDelegatedRequestFinished{}

func (m *VDelegatedRequestFinished) validateUnimplemented() error {
	switch {
	case !m.ResultFlags.IsEmpty():
		return throw.New("ResultFlags should be empty")
	case !m.EntryHeadHash.IsEmpty():
		return throw.New("EntryHeadHash should be empty")
	}
	return nil
}

func (m *VDelegatedRequestFinished) isIntolerable() bool {
	return m.GetCallFlags().GetInterference() == isolation.CallIntolerable
}

func (m *VDelegatedRequestFinished) Validate(currentPulse pulse.Number) error {
	if err := m.validateUnimplemented(); err != nil {
		return err
	} else if err := validCallType(m.GetCallType()); err != nil {
		return err
	}

	if !m.GetCallFlags().IsValid() {
		return throw.New("CallFlags should be valid")
	}

	calleePulse, err := validSelfScopedGlobalWithPulseSpecialOrBeforeOrEq(m.Callee.GetGlobal(), currentPulse, "Callee")
	if err != nil {
		return err
	}

	outgoingLocalPulse, err := validRequestGlobalWithPulseBeforeOrEq(m.CallOutgoing.GetGlobal(), currentPulse, "CallOutgoing")
	if err != nil {
		return err
	}

	incomingLocalPulse, err := validRequestGlobalWithPulseBeforeOrEq(m.CallIncoming.GetGlobal(), currentPulse, "CallIncoming")
	if err != nil {
		return err
	}

	switch {
	case !outgoingLocalPulse.IsEqOrAfter(incomingLocalPulse):
		return throw.New("CallOutgoing pulse should be more or equal than CallIncoming pulse")
	case !incomingLocalPulse.IsEqOrAfter(calleePulse):
		return throw.New("Callee pulse should be less or equal than CallOutgoing pulse")
	case m.isIntolerable() && m.LatestState != nil:
		return throw.New("LatestState should be empty on Intolerable call")
	case m.CallType == CallTypeConstructor && m.LatestState == nil:
		return throw.New("LatestState should be non-empty on Constructor call")
	}

	return nil
}
