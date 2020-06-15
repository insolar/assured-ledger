// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

type authSubjectMode uint8

const (
	_ authSubjectMode = iota
	hasAuthSubject
	hasAuthSubjectForPrevPulse
)

func GetSenderAuthenticationSubjectAndPulse(msg interface{}) (ref Reference, usePrevPulse, ok bool) {
	type customAuthSubject interface {
		customSubject() (Reference, authSubjectMode)
	}

	switch m := msg.(type) {
	case customAuthSubject:
		ref, mode := m.customSubject()
		switch mode {
		case hasAuthSubjectForPrevPulse:
			return ref, true, true
		case hasAuthSubject:
			return ref, false, true
		}
	case interface{ GetCaller() Reference }:
		return m.GetCaller(), false, true
	}
	return Reference{}, false, false
}

func (m *VStateRequest) customSubject() (Reference, authSubjectMode) {
	return m.GetObject(), hasAuthSubject
}

func (m *VCallResult) customSubject() (Reference, authSubjectMode) {
	return m.GetCallee(), hasAuthSubject
}

func (m *VStateReport) customSubject() (Reference, authSubjectMode) {
	return m.GetObject(), hasAuthSubjectForPrevPulse
}

func (m *VDelegatedCallRequest) customSubject() (Reference, authSubjectMode) {
	return m.GetCallee(), hasAuthSubjectForPrevPulse
}

func (m *VDelegatedCallResponse) customSubject() (Reference, authSubjectMode) {
	return m.GetCallee(), hasAuthSubject
}
