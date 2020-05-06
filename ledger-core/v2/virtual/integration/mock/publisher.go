// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package mock

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

type PublisherMock struct {
	Checker func(topic string, messages ...*message.Message) error
}

func (p *PublisherMock) Publish(topic string, messages ...*message.Message) error {
	if err := p.Checker(topic, messages...); err != nil {
		panic(err)
	}
	return nil
}

func (*PublisherMock) Close() error { return nil }
