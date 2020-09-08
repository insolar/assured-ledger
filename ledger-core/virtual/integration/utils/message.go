// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/trace"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type RequestWrapper struct {
	pulseNumber pulse.Number
	payload     rms.Marshaler

	sender   reference.Global
	receiver reference.Global
}

func NewRequestWrapper(pulseNumber pulse.Number, payload rms.Marshaler) *RequestWrapper {
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

	msg, err := rms.NewMessage(&rms.Meta{
		Payload:    payloadBytes,
		Sender:     w.sender,
		Receiver:   w.receiver,
		Pulse:      w.pulseNumber,
		ID:         nil,
		OriginHash: rms.MessageHash{},
	})
	if err != nil {
		panic(throw.W(err, "failed to create watermill message"))
	}

	msg.Metadata.Set(defaults.TraceID, trace.RandID())

	return msg
}
