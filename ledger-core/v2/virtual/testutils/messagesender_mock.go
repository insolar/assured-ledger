package testutils

import (
	"context"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	messageSender "github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender/adapter"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

type AsyncCallRequesterMock struct {
	childMock *smachine.AsyncCallRequesterMock
}

func NewAsyncCallRequesterMock(t minimock.Tester) *AsyncCallRequesterMock {
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
	role node.DynamicRole,
	object reference.Global,
	pn pulse.Number,
	_ ...messageSender.SendOption,
) error

type AsyncCallRequesterSendRoleMock struct {
	parentMock *MessageServiceMock

	checkMessageFn     func(msg payload.Marshaler)
	checkRoleFn        func(role node.DynamicRole)
	checkObjectFn      func(object reference.Global)
	checkPulseNumberFn func(pn pulse.Number)
}

func NewAsyncCallRequesterSendRoleMock(parent *MessageServiceMock) *AsyncCallRequesterSendRoleMock {
	mock := AsyncCallRequesterSendRoleMock{
		parentMock: parent,
	}
	return &mock
}

func (m *AsyncCallRequesterSendRoleMock) SetCheckMessage(fn func(payload.Marshaler)) *AsyncCallRequesterSendRoleMock {
	m.parentMock.Mock().SendRoleMock.Set(m.check)
	m.checkMessageFn = fn
	return m

}

func (m *AsyncCallRequesterSendRoleMock) SetCheckRole(fn func(node.DynamicRole)) *AsyncCallRequesterSendRoleMock {
	m.parentMock.Mock().SendRoleMock.Set(m.check)
	m.checkRoleFn = fn
	return m
}

func (m *AsyncCallRequesterSendRoleMock) SetCheckObject(fn func(reference.Global)) *AsyncCallRequesterSendRoleMock {
	m.parentMock.Mock().SendRoleMock.Set(m.check)
	m.checkObjectFn = fn
	return m

}

func (m *AsyncCallRequesterSendRoleMock) SetCheckPulseNumber(fn func(pulse.Number)) *AsyncCallRequesterSendRoleMock {
	m.parentMock.Mock().SendRoleMock.Set(m.check)
	m.checkPulseNumberFn = fn
	return m
}

func (m *AsyncCallRequesterSendRoleMock) check(
	_ context.Context,
	msg payload.Marshaler,
	role node.DynamicRole,
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

// ==================================================================================================

type MessageServiceMock struct {
	t         minimock.Tester
	childMock *messageSender.ServiceMock
	SendRole  *AsyncCallRequesterSendRoleMock
}

func NewMessageServiceMock(t minimock.Tester) *MessageServiceMock {
	mock := MessageServiceMock{
		t:         t,
		childMock: messageSender.NewServiceMock(t),
	}
	mock.SendRole = NewAsyncCallRequesterSendRoleMock(&mock)
	return &mock
}

func (m *MessageServiceMock) Mock() *messageSender.ServiceMock {
	return m.childMock
}

func (m *MessageServiceMock) NewAdapterMock() *MessageSenderMock {
	return NewMessageSenderMock(m.t, m.Mock())
}

// ==================================================================================================

type MessageSenderMock struct {
	t         minimock.Tester
	service   messageSender.Service
	childMock *messageSenderAdapter.MessageSenderMock
}

func NewMessageSenderMock(t minimock.Tester, svc messageSender.Service) *MessageSenderMock {
	mock := MessageSenderMock{
		t:         t,
		service:   svc,
		childMock: messageSenderAdapter.NewMessageSenderMock(t),
	}
	return &mock
}

func (m *MessageSenderMock) Mock() *messageSenderAdapter.MessageSenderMock {
	return m.childMock
}

func (m *MessageSenderMock) SetDefaultPrepareAsyncCall(ctx context.Context) *MessageSenderMock {
	cb := func(_ smachine.ExecutionContext, fn messageSenderAdapter.AsyncCallFunc) smachine.AsyncCallRequester {
		fn(ctx, m.service)
		return NewAsyncCallRequesterMock(m.t).SetWithoutAutoWakeUp().SetStart().Mock()
	}
	m.childMock.PrepareAsyncMock.Set(cb)
	return m
}

// ==================================================================================================
