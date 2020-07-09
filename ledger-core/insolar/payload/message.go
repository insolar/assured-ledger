// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

func NewMessage(pl Marshaler) (*message.Message, error) {
	buf, err := pl.Marshal()
	if err != nil {
		return nil, err
	}

	msg := message.NewMessage(watermill.NewUUID(), buf)
	// for debug logging only!
	msg.Metadata.Set("payload_string", pl.String())
	return msg, nil
}

func MustNewMessage(pl Marshaler) *message.Message {
	msg, err := NewMessage(pl)
	if err != nil {
		panic(err)
	}
	return msg
}
