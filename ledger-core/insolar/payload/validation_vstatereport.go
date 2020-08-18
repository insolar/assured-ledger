// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ Validatable = &VStateReport{}

type vStateReportValidateStatusFunc func(pulse.Number, pulse.Number) error

func (m *VStateReport) getValidateStatusFunc(s VStateReport_StateStatus) (vStateReportValidateStatusFunc, bool) {
	switch s {
	case Ready:
		return m.validateStatusReady, true
	case Empty:
		return m.validateStatusEmpty, true
	case Missing, Inactive:
		return m.validateStatusMissingOrInactive, true
	}

	return nil, false
}

func (m *VStateReport) Validate(currentPulse pulse.Number) error {
	if err := m.validateUnimplemented(); err != nil {
		return err
	}

	if !isTimePulseBefore(m.GetAsOf(), currentPulse) {
		return throw.New("AsOf should be time pulse and less that current pulse")
	}

	objectPulse, err := validSelfScopedGlobalWithPulseBeforeOrEq(m.Object, currentPulse, "Object")
	if err != nil {
		return err
	}

	if validateStatusFunc, ok := m.getValidateStatusFunc(m.GetStatus()); !ok {
		return throw.New("Unexpected state received")
	} else {
		if err := validateStatusFunc(objectPulse, currentPulse); err != nil {
			return err
		}
	}

	return nil
}

func (m *VStateReport) validateStatusReady(objectPulse pulse.Number, currentPulse pulse.Number) error {
	switch pendingCount, earliestPendingPulse := m.GetUnorderedPendingCount(), m.GetUnorderedPendingEarliestPulse(); {
	case pendingCount == 0:
		if !earliestPendingPulse.IsUnknown() {
			return throw.New("UnorderedPendingEarliestPulse should be Unknown")
		}
	case pendingCount > 0 && pendingCount < 127:
		if !isTimePulseBeforeOrEq(earliestPendingPulse, currentPulse) || !earliestPendingPulse.IsEqOrAfter(objectPulse) {
			return throw.New("UnorderedPendingEarliestPulse should be in range [objectPulse..currentPulse]")
		}
	default:
		return throw.New("UnorderedPendingCount should be in range [0..127)")
	}

	switch pendingCount, earliestPendingPulse := m.GetOrderedPendingCount(), m.GetOrderedPendingEarliestPulse(); {
	case pendingCount == 0:
		if !earliestPendingPulse.IsUnknown() {
			return throw.New("UnorderedPendingEarliestPulse should be Unknown")
		}
	case pendingCount > 0 && pendingCount < 127:
		if !isTimePulseBeforeOrEq(earliestPendingPulse, currentPulse) || !earliestPendingPulse.IsEqOrAfter(objectPulse) {
			return throw.New("OrderedPendingEarliestPulse should be in range [objectPulse..currentPulse]")
		}
	default:
		return throw.New("UnorderedPendingCount should be in range [0..127)")
	}

	return nil
}

func (m *VStateReport) validateStatusEmpty(objectPulse pulse.Number, currentPulse pulse.Number) error {
	if m.GetOrderedPendingCount() != 1 {
		return throw.New("Should be one ordered pending")
	}

	if m.GetUnorderedPendingCount() != 0 {
		return throw.New("Unordered pending count should be 0")
	}

	if !m.GetUnorderedPendingEarliestPulse().IsUnknown() {
		return throw.New("Unordered pending earliest pulse should be Unknown")
	}

	var (
		orderedPendingPulse = m.GetOrderedPendingEarliestPulse()
	)

	if !isTimePulseBefore(orderedPendingPulse, currentPulse) || !objectPulse.IsBeforeOrEq(orderedPendingPulse) {
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

func (m *VStateReport) validateStatusMissingOrInactive(pulse.Number, pulse.Number) error {
	// validate we've got zero pendings on object
	switch {
	case m.GetOrderedPendingCount() != 0:
		return throw.New("OrderedPendingCount should be 0")
	// TODO: PLAT-717: VStateReport can be Inactive and contain UnorderedPendingCount > 0 in R0
	case m.GetStatus() == Missing && m.GetUnorderedPendingCount() != 0:
		return throw.New("UnorderedPendingCount should be 0")
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
	case !m.GetPreRegisteredEarliestPulse().IsUnknown():
		return throw.New("PreRegisteredEarliestPulse should be unknown")
	case m.GetPriorityCallQueueCount() != 0:
		return throw.New("PriorityCallQueueCount should be 0")
	}

	if err := m.ProvidedContent.validateUnimplemented(); err != nil {
		return err
	}

	return nil
}

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
		}
	}

	return nil
}

func (m *VStateReport_ProvidedContentBody) validateUnimplemented() error {
	if m != nil {
		switch {
		case len(m.GetOrderedQueue()) != 0:
			return throw.New("ProvidedContent.OrderedQueue should be empty")
		case len(m.GetUnorderedQueue()) != 0:
			return throw.New("ProvidedContent.UnorderedQueue should be empty")
		}
	}

	return nil
}
