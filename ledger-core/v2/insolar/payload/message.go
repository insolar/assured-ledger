// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

func NewMessage(pl Payload) (*message.Message, error) {
	buf, err := Marshal(pl)
	if err != nil {
		return nil, err
	}
	return message.NewMessage(watermill.NewUUID(), buf), nil
}

func MustNewMessage(pl Payload) *message.Message {
	msg, err := NewMessage(pl)
	if err != nil {
		panic(err)
	}
	return msg
}
