// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package checker

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Sender interface {
	SendMessage(context.Context, *message.Message)
}

type Checker interface {
	CheckMessages(topic string, messages ...*message.Message) error
}

var _ Checker = &Default{}

type CallbackFn func(topic string, messages ...*message.Message) error

type Default struct {
	timeout   time.Duration
	checkerFn CallbackFn
}

func (p *Default) CheckMessages(topic string, messages ...*message.Message) error {
	var (
		done = make(synckit.ClosableSignalChannel)
		err  error
	)

	go func() {
		defer func() { _ = synckit.SafeClose(done) }()

		err = p.checkerFn(topic, messages...)
	}()

	select {
	case <-done:
	case <-time.After(p.timeout):
		err = throw.New("timeout: failed to check message")
	}

	return err
}

func NewDefault(fn CallbackFn) *Default {
	return &Default{
		timeout:   10 * time.Second,
		checkerFn: fn,
	}
}

func NewResend(ctx context.Context, sender Sender) *Default {
	return NewDefault(func(topic string, messages ...*message.Message) error {
		for _, msg := range messages {
			sender.SendMessage(ctx, msg)
		}
		return nil
	})
}
