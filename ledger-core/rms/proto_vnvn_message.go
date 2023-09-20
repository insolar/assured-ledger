package rms

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

func NewMessage(pl Marshaler) (*message.Message, error) {
	buf, err := pl.Marshal()
	if err != nil {
		return nil, err
	}
	return message.NewMessage(watermill.NewUUID(), buf), nil
}

func MustNewMessage(pl Marshaler) *message.Message {
	msg, err := NewMessage(pl)
	if err != nil {
		panic(err)
	}
	return msg
}
