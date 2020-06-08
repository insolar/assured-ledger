// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package mock

import (
	"context"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/gojuno/minimock"
)

var _ message.Publisher = &PublisherMock{}

type CheckerFn func(topic string, messages ...*message.Message) error

type Checker interface {
	CheckMessages(topic string, messages ...*message.Message) error
}

type PublisherMock struct {
	lock            sync.RWMutex
	messageNotifier chan struct{}
	messageCounter  int
	checker         Checker
	closed          bool
}

type Sender interface {
	SendMessage(context.Context, *message.Message)
}

func NewPublisherMock() *PublisherMock {
	return &PublisherMock{
		messageNotifier: make(chan struct{}, 1),
	}
}

func (p *PublisherMock) messageCount() int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.messageCounter
}

func (p *PublisherMock) messageCountUpdate(count int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.messageCounter += count
}

func (p *PublisherMock) GetCount() int {
	return p.messageCounter
}

func (p *PublisherMock) WaitCount(count int, timeout time.Duration) bool {
	for {
		if p.messageCount() >= count {
			return true
		}
		started := time.Now()
		select {
		case <-p.messageNotifier:
			timeout -= time.Since(started)
		case <-time.After(timeout):
			return false
		}
	}
}

func (p *PublisherMock) SetResenderMode(ctx context.Context, sender Sender) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.checker = NewResenderPublishChecker(ctx, sender)
}

func (p *PublisherMock) SetTypedChecker(ctx context.Context, t minimock.Tester, sender Sender) *TypePublishChecker {
	p.lock.Lock()
	defer p.lock.Unlock()

	checker := NewTypePublishChecker(ctx, t, sender)
	p.checker = checker

	return checker
}

func (p *PublisherMock) SetBaseChecker(fn Checker) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.checker = fn
}

func (p *PublisherMock) SetChecker(fn CheckerFn) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.checker = NewDefaultPublishChecker(fn)
}

func (p *PublisherMock) getChecker() CheckerFn {
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

func (p *PublisherMock) Publish(topic string, messages ...*message.Message) error {
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

func (p *PublisherMock) notify() {
	select {
	case p.messageNotifier <- struct{}{}:
	default:
	}
}

func (p *PublisherMock) Close() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.closed = true

	return nil
}
