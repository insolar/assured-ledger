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

	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/mock/publisher/checker"
)

// go:generate insolar-mockgenerator-typedpublisher

var _ message.Publisher = &Mock{}

type Mock struct {
	lock            sync.RWMutex
	messageNotifier chan struct{}
	messageCounter  int
	checker         checker.Checker
	closed          bool
}

type Sender interface {
	SendMessage(context.Context, *message.Message)
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

func (p *Mock) SetResendMode(ctx context.Context, sender Sender) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.checker = checker.NewResend(ctx, sender)
}

func (p *Mock) SetTypedChecker(ctx context.Context, t minimock.Tester, sender Sender) *checker.Typed {
	p.lock.Lock()
	defer p.lock.Unlock()

	c := checker.NewTyped(ctx, t, sender)
	p.checker = c

	return c
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
