// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

type AuthSubjectMode uint8

const (
	_ AuthSubjectMode = iota
	UseCurrentPulse
	UsePrevPulse
	UseAnyPulse
)

func GetSenderAuthenticationSubjectAndPulse(msg interface{}) (ref Reference, mode AuthSubjectMode, ok bool) {
	type customAuthSubject interface {
		customSubject() (Reference, AuthSubjectMode)
	}

	switch m := msg.(type) {
	case customAuthSubject:
		ref, mode := m.customSubject()
		return ref, mode, true
	case interface{ GetCaller() Reference }:
		return m.GetCaller(), UseCurrentPulse, true
	}
	return Reference{}, UseCurrentPulse, false
}

func (m *VStateRequest) customSubject() (Reference, AuthSubjectMode) {
	return m.GetObject(), UseCurrentPulse
}

func (m *VCallResult) customSubject() (Reference, AuthSubjectMode) {
	return m.GetCallee(), UseCurrentPulse
}

func (m *VStateReport) customSubject() (Reference, AuthSubjectMode) {
	return m.GetObject(), UsePrevPulse
}

func (m *VDelegatedCallRequest) customSubject() (Reference, AuthSubjectMode) {
	return m.GetCallee(), UsePrevPulse
}

func (m *VDelegatedCallResponse) customSubject() (Reference, AuthSubjectMode) {
	return m.GetCallee(), UseCurrentPulse
}

// customSubject returns hasAuthSubject ( i.e. for current pulse ) since VDelegatedRequestFinished can come only with
// delegation token and subject is going to be considered as Approver of this delegation token
func (m *VDelegatedRequestFinished) customSubject() (Reference, AuthSubjectMode) {
	return m.GetCallee(), UseCurrentPulse
}

func (m *VFindCallRequest) customSubject() (Reference, AuthSubjectMode) {
	return m.GetCallee(), UseCurrentPulse
}

func (m *VFindCallResponse) customSubject() (Reference, AuthSubjectMode) {
	return m.GetCallee(), UseAnyPulse
}

func (m *VCachedMemoryRequest) customSubject() (Reference, AuthSubjectMode) {
	return m.GetObject(), UseCurrentPulse
}

func (m *VObjectValidationReport) customSubject() (Reference, AuthSubjectMode) {
	return m.GetObject(), UsePrevPulse
}
