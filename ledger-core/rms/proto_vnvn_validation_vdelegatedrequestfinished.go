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
	case m.ResultFlags != 0:
		return throw.New("ResultFlags should be zero")
	default:
		return nil
	}
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

	calleePulse, err := validSelfScopedGlobalWithPulseSpecialOrBeforeOrEq(m.Callee.GetValue(), currentPulse, "Callee")
	if err != nil {
		return err
	}

	outgoingLocalPulse, err := validRequestGlobalWithPulseBeforeOrEq(m.CallOutgoing.GetValue(), currentPulse, "CallOutgoing")
	if err != nil {
		return err
	}

	incomingLocalPulse, err := validRequestGlobalWithPulseBeforeOrEq(m.CallIncoming.GetValue(), currentPulse, "CallIncoming")
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

	transcript := m.GetPendingTranscript()
	if len(transcript.GetEntries()) == 0 {
		return throw.New("PendingTranscript mustn't be empty")
	}

	transcriptErr := validateEntries(transcript.GetEntries())
	if transcriptErr != nil {
		return throw.W(transcriptErr, "PendingTranscript validation failed")
	}

	return nil
}
