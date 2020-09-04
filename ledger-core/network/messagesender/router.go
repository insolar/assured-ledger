// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package messagesender

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
)

type MessageRouter interface {
	CreateMessageSender(affinity.Helper, beat.History) Service
	SubscribeForMessages(func(beat.Message) error) (stopFn func())
	PublishMessage(topic string, msg *message.Message) error
}
