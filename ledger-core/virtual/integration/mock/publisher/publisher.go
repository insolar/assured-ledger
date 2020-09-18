// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package publisher

import (
	"context"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/gojuno/minimock/v3"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/mock/publisher/checker"
)

//go:generate insolar-mockgenerator-typedpublisher

var _ message.Publisher = &Mock{}

type Mock struct {
	lock            sync.RWMutex
	messageNotifier chan struct{}
	messageCounter  int
	checker         checker.Checker
	closed          bool
}

type MinimalSender interface {
	SendMessage(context.Context, *message.Message)
}

type Sender interface {
	MinimalSender
	SendPayload(ctx context.Context, pl rmsreg.GoGoSerializable)
}

func NewMock() *Mock {
	return &Mock{
		messageNotifier: make(chan struct{}, 1),
	}
}

func (p *Mock) GetCount() int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.messageCounter
}

func (p *Mock) messageCountUpdate(count int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.messageCounter += count
}

func (p *Mock) WaitCount(count int, timeout time.Duration) bool {
	timeoutCh := time.After(timeout)
	for {
		if p.GetCount() >= count {
			return true
		}
		select {
		case <-p.messageNotifier:
		case <-timeoutCh:
			return false
		}
	}
}

func (p *Mock) SetResendMode(ctx context.Context, sender MinimalSender) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.checker = checker.NewResend(ctx, sender)
}

func (p *Mock) SetTypedChecker(ctx context.Context, t minimock.Tester, sender Sender) *checker.Typed {
	p.lock.Lock()
	defer p.lock.Unlock()

	typed := checker.NewTyped(ctx, t, sender)
	p.checker = typed

	return typed
}

func (p *Mock) SetTypedCheckerWithLightStubs(ctx context.Context, t minimock.Tester, sender Sender) *checker.Typed {
	p.lock.Lock()
	defer p.lock.Unlock()

	typed := checker.NewTyped(ctx, t, sender)
	p.checker = typed

	logger := inslogger.FromContext(ctx)

	typed.LRegisterRequest.Set(func(request *rms.LRegisterRequest) bool {
		if request.Flags == rms.RegistrationFlags_FastSafe {
			logger.Info("Sending Fast and Safe")

			sender.SendPayload(ctx, &rms.LRegisterResponse{
				Flags:              rms.RegistrationFlags_Fast,
				AnticipatedRef:     request.AnticipatedRef,
				RegistrarSignature: rms.NewBytes([]byte("123")),
			})

			time.Sleep(10 * time.Millisecond)

			sender.SendPayload(ctx, &rms.LRegisterResponse{
				Flags:              rms.RegistrationFlags_Safe,
				AnticipatedRef:     request.AnticipatedRef,
				RegistrarSignature: rms.NewBytes([]byte("123")),
			})
		} else {
			logger.Info("Sending ", request.Flags.String())

			sender.SendPayload(ctx, &rms.LRegisterResponse{
				Flags:              request.Flags,
				AnticipatedRef:     request.AnticipatedRef,
				RegistrarSignature: rms.NewBytes([]byte("123")),
			})
		}

		return false
	})

	return typed
}

func (p *Mock) SetBaseChecker(fn checker.Checker) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.checker = fn
}

func (p *Mock) SetChecker(fn checker.CallbackFn) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.checker = checker.NewDefault(fn)
}

func (p *Mock) getChecker() checker.CallbackFn {
	p.lock.RLock()
	defer p.lock.RUnlock()

	if p.closed {
		panic("publisher is closed")
	}

	if p.checker == nil {
		panic("checker function is empty")
	}

	return p.checker.CheckMessages
}

func (p *Mock) Publish(topic string, messages ...*message.Message) error {
	defer func() {
		p.messageCountUpdate(len(messages))
		p.notify()
	}()

	fn := p.getChecker()
	if err := fn(topic, messages...); err != nil {
		panic(err)
	}
	return nil
}

func (p *Mock) notify() {
	select {
	case p.messageNotifier <- struct{}{}:
	default:
	}
}

func (p *Mock) Close() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.closed = true

	return nil
}
