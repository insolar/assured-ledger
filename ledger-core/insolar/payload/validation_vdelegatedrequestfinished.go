// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VDelegatedRequestFinished{}

func (m *VDelegatedRequestFinished) validateUnimplemented() error {
	switch {
	case m.ResultFlags != nil:
		return throw.New("ResultFlags should be empty")
	case m.EntryHeadHash != nil:
		return throw.New("EntryHeadHash should be empty")
	}
	return nil
}

func (m *VDelegatedRequestFinished) isIntolerable() bool {
	return m.GetCallFlags().GetInterference() == contract.CallIntolerable
}

func (m *VDelegatedRequestFinished) Validate(currentPulse pulse.Number) error {
	if err := m.validateUnimplemented(); err != nil {
		return err
	}

	switch m.GetCallType() {
	case CTMethod, CTConstructor:
	case CTInboundAPICall, CTOutboundAPICall, CTNotifyCall, CTSAGACall, CTParallelCall, CTScheduleCall:
		return throw.New("CallType is not implemented")
	case CTInvalid:
		fallthrough
	default:
		return throw.New("CallType must be valid")
	}

	callFlags := m.GetCallFlags()
	if f := callFlags.GetState(); f >= contract.StateFlagCount || f <= contract.StateInvalid {
		return throw.New("CallFlags state should be valid")
	} else if f := callFlags.GetInterference(); f >= contract.InterferenceFlagCount || f <= contract.InterferenceInvalid {
		return throw.New("CallFlags interference should be valid")
	}

	if !m.Callee.IsSelfScope() {
		return throw.New("Callee must be valid self scoped Reference")
	} else if pn := m.Callee.GetLocal().Pulse(); !pn.IsTimePulse() || pn >= currentPulse {
		return throw.New("Callee must have valid time pulse with pulse lesser than current pulse")
	}

	if m.CallIncoming.IsEmpty() {
		return throw.New("CallIncoming should be non-empty")
	} else if pn := m.CallIncoming.GetLocal().Pulse(); !pn.IsTimePulse() || pn >= currentPulse {
		return throw.New("CallIncoming local part should have valid time pulse lesser than current pulse")
	} else if pn := m.CallIncoming.GetBase().Pulse(); !pn.IsTimePulse() || pn >= currentPulse {
		return throw.New("CallIncoming base part should have valid time pulse lesser than current pulse")
	} else if m.CallIncoming.GetBase().Pulse() > m.CallIncoming.GetLocal().Pulse() {
		return throw.New("CallIncoming base pulse should be less or equal than local pulse")
	}

	if m.CallOutgoing.IsEmpty() {
		return throw.New("CallOutgoing should be non-empty")
	} else if pn := m.CallOutgoing.GetLocal().Pulse(); !pn.IsTimePulse() || pn >= currentPulse {
		return throw.New("CallOutgoing local part should have valid time pulse lesser than current pulse")
	} else if pn := m.CallOutgoing.GetBase().Pulse(); !pn.IsSpecialOrTimePulse() || pn >= currentPulse {
		// probably call outgoing base part can be special (API Call)
		return throw.New("CallOutgoing base part should have valid pulse lesser than current pulse")
	} else if m.CallOutgoing.GetBase().Pulse() > m.CallOutgoing.GetLocal().Pulse() {
		return throw.New("CallIncoming base pulse should be less or equal than local pulse")
	}

	if m.CallOutgoing.GetLocal().Pulse() < m.CallIncoming.GetLocal().Pulse() {
		return throw.New("CallOutgoing pulse should be more or equal than CallIncoming pulse")
	} else if m.Callee.GetLocal().Pulse() > m.CallOutgoing.GetLocal().Pulse() {
		return throw.New("Callee pulse should be less or equal than CallOutgoing pulse")
	}

	if m.isIntolerable() && m.LatestState != nil {
		return throw.New("LatestState should be empty on Intolerable call")
	} else if m.CallType == CTConstructor && m.LatestState == nil {
		return throw.New("LatestState should be non-empty on Constructor call")
	}

	return nil
}
