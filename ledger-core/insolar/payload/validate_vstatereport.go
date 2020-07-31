// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VStateReport{}

func (m *VStateReport_ProvidedContentBody) validateIsEmpty() error {
	if m != nil {
		switch {
		case m.GetLatestValidatedState() != nil:
			return throw.New("ProvidedContent.LatestValidatedState should be empty")
		case m.GetLatestValidatedCode() != nil:
			return throw.New("ProvidedContent.LatestValidatedCode should be empty")
		case m.GetLatestDirtyState() != nil:
			return throw.New("ProvidedContent.LatestDirtyState should be empty")
		case m.GetLatestDirtyCode() != nil:
			return throw.New("ProvidedContent.LatestDirtyCode should be empty")
		case len(m.GetOrderedQueue()) != 0:
			return throw.New("ProvidedContent.OrderedQueue should be empty")
		case len(m.GetUnorderedQueue()) != 0:
			return throw.New("ProvidedContent.UnorderedQueue should be empty")
		}
	}

	return nil
}

func (m *VStateReport) Validate(currPulse PulseNumber) error {
	if err := m.validateUnimplemented(); err != nil {
		return err
	}

	if asOf := m.GetAsOf(); asOf.IsTimePulse() && asOf >= currPulse {
		return throw.New("AsOf should be time pulse and less that current pulse")
	}

	if !m.GetObject().IsSelfScope() {
		return throw.New("Object reference should be self scoped")
	}

	if m.GetObject().GetLocal().Pulse() >= currPulse {
		return throw.New("Object pulse should be less that current pulse")
	}

	switch m.GetStatus() {
	case Ready:
		objectPulseNumber := m.GetObject().GetLocal().GetPulseNumber()
		if err := m.validateStatusReady(objectPulseNumber, currPulse); err != nil {
			return err
		}
	case Empty:
		if err := m.validateStatusEmpty(currPulse); err != nil {
			return err
		}
	case Missing, Inactive:
		if err := m.validateStatusMissingOrInactive(); err != nil {
			return err
		}
	default:
		return throw.New("Unexpected state received")
	}

	return nil
}

func (m *VStateReport) validateStatusEmpty(currPulse PulseNumber) error {
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

	if err := m.validateStateAndCodeAreEmpty(); err != nil {
		return err
	}

	if err := m.ProvidedContent.validateIsEmpty(); err != nil {
		return err
	}

	return nil
}

func (m *VStateReport) validateStatusReady(objectPulse PulseNumber, currPulse PulseNumber) error {
	switch pendingCount := m.GetUnorderedPendingCount(); {
	case pendingCount == 0:
		//
	case pendingCount > 0 && pendingCount < 127:
		earliestPendingPulse := m.GetUnorderedPendingEarliestPulse()
		if earliestPendingPulse < objectPulse || earliestPendingPulse > currPulse {
			return throw.New("UnorderedPendingEarliestPulse should be in range (objectPulse..currPulse]")
		}
	default:
		return throw.New("UnorderedPendingCount should be in range [0..127)")
	}

	switch pendingCount := m.GetOrderedPendingCount(); {
	case pendingCount == 0:
		//
	case pendingCount > 0 && pendingCount < 127:
		earliestPendingPulse := m.GetOrderedPendingEarliestPulse()
		if earliestPendingPulse < objectPulse || earliestPendingPulse > currPulse {
			return throw.New("OrderedPendingEarliestPulse should be in range (objectPulse..currPulse]")
		}
	default:
		return throw.New("UnorderedPendingCount should be in range [0..127)")
	}

	return nil
}

func (m *VStateReport) validateStatusMissingOrInactive() error {
	// validate we've got zero pendings on object
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

	// validate ProvidedContent is empty
	if err := m.ProvidedContent.validateIsEmpty(); err != nil {
		return err
	}

	// validate internal state description is empty
	if err := m.validateStateAndCodeAreEmpty(); err != nil {
		return err
	}

	return nil
}

func (m *VStateReport) validateStateAndCodeAreEmpty() error {
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
