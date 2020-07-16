// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package checker

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
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
	checkerFn CallbackFn
}

func (p *Default) CheckMessages(topic string, messages ...*message.Message) error {
	return p.checkerFn(topic, messages...)
}

func NewDefault(fn CallbackFn) *Default {
	return &Default{
		checkerFn: fn,
	}
}

func NewResend(ctx context.Context, sender Sender) *Default {
	return &Default{
		checkerFn: func(topic string, messages ...*message.Message) error {
			for _, msg := range messages {
				sender.SendMessage(ctx, msg)
			}
			return nil
		},
	}
}
