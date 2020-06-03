// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/trace"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type RequestWrapper struct {
	pulseNumber pulse.Number
	payload     payload.Marshaler

	sender   reference.Global
	receiver reference.Global
}

func NewRequestWrapper(pulseNumber pulse.Number, payload payload.Marshaler) *RequestWrapper {
	return &RequestWrapper{
		pulseNumber: pulseNumber,
		payload:     payload,
	}
}

func (w *RequestWrapper) SetSender(sender reference.Global) *RequestWrapper {
	w.sender = sender
	return w
}

func (w *RequestWrapper) SetReceiver(receiver reference.Global) *RequestWrapper {
	w.receiver = receiver
	return w
}

func (w *RequestWrapper) Finalize() *message.Message {
	payloadBytes, err := w.payload.Marshal()
	if err != nil {
		panic(throw.W(err, "failed to marshal message"))
	}

	msg, err := payload.NewMessage(&payload.Meta{
		Payload:    payloadBytes,
		Sender:     w.sender,
		Receiver:   w.receiver,
		Pulse:      w.pulseNumber,
		ID:         nil,
		OriginHash: payload.MessageHash{},
	})
	if err != nil {
		panic(throw.W(err, "failed to create watermill message"))
	}

	msg.Metadata.Set(defaults.TraceID, trace.RandID())

	return msg
}
