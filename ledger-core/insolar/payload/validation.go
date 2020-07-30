// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Validate interface {
	Validate(currPulse PulseNumber) error
}

var _ Validate = &VStateReport{}

func (m *VStateReport) Validate(currPulse PulseNumber) error {
	if err := m.validateUnimplemented(); err != nil {
		return err
	}

	if m.GetAsOf().IsUnknown() {
		return throw.New("AsOf can not be Unknown")
	}

	if m.GetAsOf() >= currPulse {
		return throw.New("AsOf should be less than current pulse")
	}

	if m.GetObject().IsEmpty() {
		return throw.New("Object should not be empty")
	}

	if !m.GetObject().IsSelfScope() {
		return throw.New("Object reference should be selfScore")
	}

	if m.GetObject().GetLocal().Pulse() >= currPulse {
		return throw.New("Object pulse should be less that current pulse")
	}

	switch m.GetStatus() {
	case Unknown:
		return throw.New("Unexepected Unknown status received")
	case Ready:
		if err := validateReadyStatus(m, currPulse); err != nil {
			return err
		}
	case Empty:
		if err := validateEmptyStatus(m, currPulse); err != nil {
			return err
		}
	case Missing, Inactive:
		if err := validateZeroPending(m); err != nil {
			return err
		}

		if err := validateEmptyState(m); err != nil {
			return err
		}

		if err := validateEmptyLatest(m); err != nil {
			return err
		}

	}

	return nil
}

func validateEmptyStatus(m *VStateReport, currPulse PulseNumber) error {
	if m.GetOrderedPendingCount() != 1 {
		return throw.New("Should be one ordered pending")
	}

	if m.GetUnorderedPendingCount() != 0 {
		return throw.New("Unordered pending count should be 0")
	}

	if !m.GetUnorderedPendingEarliestPulse().IsUnknown() {
		return throw.New("Unordered pending earliest pulse should be Unknown")
	}

	objectPulse := m.GetAsOf()
	orderedPendingPulse := m.GetOrderedPendingEarliestPulse()

	if orderedPendingPulse < objectPulse || currPulse < orderedPendingPulse {
		return throw.New("Incorrect pending ordered pulse number")
	}

	return nil
}

func validateReadyStatus(m *VStateReport, currPulse PulseNumber) error {
	if m.GetUnorderedPendingCount() < 0 || m.GetUnorderedPendingCount() >= 127 {
		return throw.New("UnorderedPendingCount should be in range [0..127)")
	}

	if m.GetOrderedPendingCount() < 0 || m.GetOrderedPendingCount() >= 127 {
		return throw.New("OrderedPendingCount should be in range [0..127)")
	}

	if m.GetUnorderedPendingEarliestPulse() < 0 || m.GetUnorderedPendingEarliestPulse() > currPulse {
		return throw.New("UnorderedPendingEarliestPulse should be in range (0..currPulse]")
	}

	if m.GetOrderedPendingEarliestPulse() < 0 || m.GetOrderedPendingEarliestPulse() > currPulse {
		return throw.New("OrderedPendingEarliestPulse should be in range (0..currPulse]")
	}

	return nil
}

func validateZeroPending(m *VStateReport) error {
	switch {
	case m.GetOrderedPendingCount() != 0:
		return throw.New("OrderedPendingCount should be o")
	case m.GetUnorderedPendingCount() != 0:
		return throw.New("UnorderedPendingCount should be o")
	case !m.GetOrderedPendingEarliestPulse().IsUnknown():
		return throw.New("OrderedPendingEarliestPulse should be Unknown")
	case !m.GetUnorderedPendingEarliestPulse().IsUnknown():
		return throw.New("UnorderedPendingEarliestPulse should be Unknown")
	}

	return nil
}

func validateEmptyState(m *VStateReport) error {
	switch content := m.ProvidedContent; {
	case content == nil:
		return nil
	case content.GetLatestValidatedState() != nil:
		return throw.New("ProvidedContent.LatestValidatedState should be empty")
	case content.GetLatestValidatedCode() != nil:
		return throw.New("ProvidedContent.LatestValidatedCode should be empty")
	case content.GetLatestDirtyState() != nil:
		return throw.New("ProvidedContent.LatestDirtyState should be empty")
	case content.GetLatestDirtyCode() != nil:
		return throw.New("ProvidedContent.LatestDirtyCode should be empty")
	case len(content.GetOrderedQueue()) != 0:
		return throw.New("ProvidedContent.OrderedQueue should be empty")
	case len(content.GetUnorderedQueue()) != 0:
		return throw.New("ProvidedContent.UnorderedQueue should be empty")
	}

	return nil
}

func validateEmptyLatest(m *VStateReport) error {
	switch {
	case !m.GetLatestValidatedState().IsEmpty():
		return throw.New("LatestValidatedState should be empty")
	case !m.GetLatestValidatedCode().IsEmpty():
		return throw.New("LatestValidatedCode should be empty")
	case !m.GetLatestDirtyState().IsEmpty():
		return throw.New("LatestDirtyState should be empty")
	case !m.GetLatestDirtyCode().IsEmpty():
		return throw.New("LatestDirtyCode should be empty")
	}

	return nil
}

func (m *VStateReport) validateUnimplemented() error {
	switch {
	case !m.GetDelegationSpec().IsZero():
		return throw.New("DelegationSpec should be empty")
	case m.GetPreRegisteredQueueCount() != 0:
		return throw.New("PriorityCallQueueCount should be 0")
	case !m.PreRegisteredEarliestPulse.IsUnknown():
		return throw.New("PreRegisteredEarliestPulse should be unknown")
	case m.GetPriorityCallQueueCount() != 0:
		return throw.New("PriorityCallQueueCount should be 0")
	}

	return nil
}
