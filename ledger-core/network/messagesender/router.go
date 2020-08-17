// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package messagesender

import (
	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
)

type MessageRouter interface {
	CreateMessageSender(affinity.Helper, beat.Accessor) Service
	SubscribeForMessages(func(beat.Message) error) (stopFn func())
}
