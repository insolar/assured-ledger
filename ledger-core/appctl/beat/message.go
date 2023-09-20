package beat

import (
	"github.com/insolar/assured-ledger/ledger-core/vanilla/synckit"
)

type MessageAcker interface {
	Ack() bool
	Acked() <-chan struct{}
}

func NewMessage(uuid string, payload []byte) Message {
	return Message{UUID: uuid, Payload: payload }
}

func NewMessageExt(uuid string, payload []byte, ack MessageAcker) Message {
	return Message{UUID: uuid, Payload: payload, ack: ack }
}

type Message struct {
	UUID string
	Metadata map[string]string
	Payload  []byte

	ack   MessageAcker
}

func (v Message) Ack() bool {
	if v.ack != nil {
		return v.ack.Ack()
	}
	return true
}

func (v Message) Acked() synckit.SignalChannel {
	if v.ack != nil {
		return v.ack.Acked()
	}
	return synckit.ClosedChannel()
}

