// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package messagesender

import (
	"context"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	messageSender "github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	payload "github.com/insolar/assured-ledger/ledger-core/rms"
)

type AsyncCallRequesterMock struct {
	childMock *smachine.AsyncCallRequesterMock
}

func NewAsyncCallRequesterMock(t *minimock.Controller) *AsyncCallRequesterMock {
	mock := AsyncCallRequesterMock{
		childMock: smachine.NewAsyncCallRequesterMock(t),
	}
	return &mock
}

func (m *AsyncCallRequesterMock) Mock() *smachine.AsyncCallRequesterMock {
	return m.childMock
}

func (m *AsyncCallRequesterMock) SetStart() *AsyncCallRequesterMock {
	m.childMock.StartMock.Return()
	return m
}

func (m *AsyncCallRequesterMock) SetWithoutAutoWakeUp() *AsyncCallRequesterMock {
	m.childMock.WithoutAutoWakeUpMock.Return(m.childMock)
	return m
}

// ==================================================================================================

type SendRoleFn func(
	_ context.Context,
	msg payload.Marshaler,
	role affinity.DynamicRole,
	object reference.Global,
	pn pulse.Number,
	_ ...messageSender.SendOption,
) error

type AsyncCallRequesterSendRoleMock struct {
	parentMock *ServiceMockWrapper

	checkMessageFn     func(msg payload.Marshaler)
	checkRoleFn        func(role affinity.DynamicRole)
	checkObjectFn      func(object reference.Global)
	checkPulseNumberFn func(pn pulse.Number)
}

func NewAsyncCallRequesterSendRoleMock(parent *ServiceMockWrapper) *AsyncCallRequesterSendRoleMock {
	mock := AsyncCallRequesterSendRoleMock{
		parentMock: parent,
	}
	return &mock
}

func (m *AsyncCallRequesterSendRoleMock) SetCheckMessage(fn func(payload.Marshaler)) *AsyncCallRequesterSendRoleMock {
	m.prepare()
	m.checkMessageFn = fn
	return m

}

func (m *AsyncCallRequesterSendRoleMock) SetCheckRole(fn func(affinity.DynamicRole)) *AsyncCallRequesterSendRoleMock {
	m.prepare()
	m.checkRoleFn = fn
	return m
}

func (m *AsyncCallRequesterSendRoleMock) SetCheckObject(fn func(reference.Global)) *AsyncCallRequesterSendRoleMock {
	m.prepare()
	m.checkObjectFn = fn
	return m

}

func (m *AsyncCallRequesterSendRoleMock) SetCheckPulseNumber(fn func(pulse.Number)) *AsyncCallRequesterSendRoleMock {
	m.prepare()
	m.checkPulseNumberFn = fn
	return m
}

func (m *AsyncCallRequesterSendRoleMock) check(
	_ context.Context,
	msg payload.Marshaler,
	role affinity.DynamicRole,
	object reference.Global,
	pn pulse.Number,
	_ ...messageSender.SendOption,
) error {
	if m.checkMessageFn != nil {
		m.checkMessageFn(msg)
	}
	if m.checkObjectFn != nil {
		m.checkObjectFn(object)
	}
	if m.checkRoleFn != nil {
		m.checkRoleFn(role)
	}
	if m.checkPulseNumberFn != nil {
		m.checkPulseNumberFn(pn)
	}

	return nil
}

func (m *AsyncCallRequesterSendRoleMock) Set(fn SendRoleFn) *AsyncCallRequesterSendRoleMock {
	m.parentMock.Mock().SendRoleMock.Set(fn)
	return m
}

func (m *AsyncCallRequesterSendRoleMock) prepare() {
	m.parentMock.Mock().SendRoleMock.Set(m.check)
}

// ==================================================================================================

type SendTargetFn func(
	_ context.Context,
	msg payload.Marshaler,
	target reference.Global,
	_ ...messageSender.SendOption,
) error

type AsyncCallRequesterSendTargetMock struct {
	parentMock *ServiceMockWrapper

	checkMessageFn func(msg payload.Marshaler)
	checkTargetFn  func(target reference.Global)
}

func NewAsyncCallRequesterSendTargetMock(parent *ServiceMockWrapper) *AsyncCallRequesterSendTargetMock {
	mock := AsyncCallRequesterSendTargetMock{
		parentMock: parent,
	}
	return &mock
}

func (m *AsyncCallRequesterSendTargetMock) SetCheckMessage(fn func(payload.Marshaler)) *AsyncCallRequesterSendTargetMock {
	m.prepare()
	m.checkMessageFn = fn
	return m

}

func (m *AsyncCallRequesterSendTargetMock) SetCheckTarget(fn func(reference.Global)) *AsyncCallRequesterSendTargetMock {
	m.prepare()
	m.checkTargetFn = fn
	return m
}
func (m *AsyncCallRequesterSendTargetMock) check(
	_ context.Context,
	msg payload.Marshaler,
	target reference.Global,
	_ ...messageSender.SendOption,
) error {
	if m.checkMessageFn != nil {
		m.checkMessageFn(msg)
	}
	if m.checkTargetFn != nil {
		m.checkTargetFn(target)
	}

	return nil
}

func (m *AsyncCallRequesterSendTargetMock) Set(fn SendTargetFn) *AsyncCallRequesterSendTargetMock {
	m.parentMock.Mock().SendTargetMock.Set(fn)
	return m
}

func (m *AsyncCallRequesterSendTargetMock) prepare() {
	m.parentMock.Mock().SendTargetMock.Set(m.check)
}

// ==================================================================================================

type ServiceMockWrapper struct {
	t         *minimock.Controller
	childMock *messageSender.ServiceMock

	SendRole   *AsyncCallRequesterSendRoleMock
	SendTarget *AsyncCallRequesterSendTargetMock
}

func NewServiceMockWrapper(t *minimock.Controller) *ServiceMockWrapper {
	mock := ServiceMockWrapper{
		t:         t,
		childMock: messageSender.NewServiceMock(t),
	}
	mock.SendRole = NewAsyncCallRequesterSendRoleMock(&mock)
	mock.SendTarget = NewAsyncCallRequesterSendTargetMock(&mock)
	return &mock
}

func (m *ServiceMockWrapper) Mock() *messageSender.ServiceMock {
	return m.childMock
}

func (m *ServiceMockWrapper) NewAdapterMock() *AdapterMockWrapper {
	return NewAdapterMockWrapper(m.t, m.Mock())
}
