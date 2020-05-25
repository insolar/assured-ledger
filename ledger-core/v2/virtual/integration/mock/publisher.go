// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package mock

import (
	"sync"

	"github.com/ThreeDotsLabs/watermill/message"
)

var _ message.Publisher = &PublisherMock{}

type CheckerFn func(topic string, messages ...*message.Message) error

type PublisherMock struct {
	lock    sync.Mutex
	checker CheckerFn
	closed  bool
}

func (p *PublisherMock) SetChecker(fn CheckerFn) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.checker = fn
}

func (p *PublisherMock) getChecker() CheckerFn {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.closed {
		panic("publisher is closed")
	}

	if p.checker == nil {
		panic("checker function is empty")
	}

	return p.checker
}

func (p *PublisherMock) Publish(topic string, messages ...*message.Message) error {
	fn := p.getChecker()
	if err := fn(topic, messages...); err != nil {
		panic(err)
	}
	return nil
}

func (p *PublisherMock) Close() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.closed = true

	return nil
}
