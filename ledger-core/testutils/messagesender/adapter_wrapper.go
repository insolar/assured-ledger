package messagesender

import (
	"context"

	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	messageSender "github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	messageSenderAdapter "github.com/insolar/assured-ledger/ledger-core/network/messagesender/adapter"
)

type AdapterMockWrapper struct {
	t         *minimock.Controller
	service   messageSender.Service
	childMock *messageSenderAdapter.MessageSenderMock
}

func NewAdapterMockWrapper(t *minimock.Controller, svc messageSender.Service) *AdapterMockWrapper {
	mock := AdapterMockWrapper{
		t:         t,
		service:   svc,
		childMock: messageSenderAdapter.NewMessageSenderMock(t),
	}
	return &mock
}

func (m *AdapterMockWrapper) Mock() *messageSenderAdapter.MessageSenderMock {
	return m.childMock
}

func (m *AdapterMockWrapper) SetDefaultPrepareAsyncCall(ctx context.Context) *AdapterMockWrapper {
	cb := func(_ smachine.ExecutionContext, fn messageSenderAdapter.AsyncCallFunc) smachine.AsyncCallRequester {
		fn(ctx, m.service)
		return NewAsyncCallRequesterMock(m.t).SetWithoutAutoWakeUp().SetStart().Mock()
	}
	m.childMock.PrepareAsyncMock.Set(cb)
	return m
}
