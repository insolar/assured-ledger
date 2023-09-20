package messagesender

import (
	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
)

type MessageRouter interface {
	CreateMessageSender(affinity.Helper, beat.History) Service
	SubscribeForMessages(func(beat.Message) error) (stopFn func())
}
