// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package messagesender

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/instracer"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type options struct {
	syncBody bool
}

// func (o *options) applyOptions(opts ...SendOption) {
// 	for _, opt := range opts {
// 		opt(o)
// 	}
// }

type SendOption func(*options)

// nolint:unused
func WithSyncBody() SendOption {
	return func(o *options) {
		o.syncBody = true
	}
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/network/messagesender.Service -o ./ -s _mock.go -g

type Service interface {
	// blocks if network unreachable
	SendRole(ctx context.Context, msg payload.Marshaler, role affinity.DynamicRole, object reference.Global, pn pulse.Number, opts ...SendOption) error
	SendTarget(ctx context.Context, msg payload.Marshaler, target reference.Global, opts ...SendOption) error
}

type DefaultService struct {
	pub      message.Publisher
	affinity affinity.Helper
	pulses   beat.History
}

func NewDefaultService(pub message.Publisher, affinity affinity.Helper, pulses beat.History) *DefaultService {
	return &DefaultService{
		pub:      pub,
		affinity: affinity,
		pulses:   pulses,
	}
}

func (dm *DefaultService) Close() error {
	return dm.pub.Close()
}

func (dm *DefaultService) SendRole(ctx context.Context, msg payload.Marshaler, role affinity.DynamicRole, object reference.Global, pn pulse.Number, opts ...SendOption) error {
	waterMillMsg, err := payload.NewMessage(msg.(payload.Marshaler))
	if err != nil {
		return throw.W(err, "Can't create watermill message")
	}

	nodes, err := dm.affinity.QueryRole(role, object, pn)
	if err != nil {
		return throw.W(err, "failed to calculate role")
	}

	if nodes[0].Equal(dm.affinity.Me()) {
		inslogger.FromContext(ctx).Debug("Send to myself")
	}

	return dm.sendTarget(ctx, waterMillMsg, nodes[0])
}

func (dm *DefaultService) SendTarget(ctx context.Context, msg payload.Marshaler, target reference.Global, opts ...SendOption) error {
	waterMillMsg, err := payload.NewMessage(msg.(payload.Marshaler))
	if err != nil {
		return throw.W(err, "Can't create watermill message")
	}

	return dm.sendTarget(ctx, waterMillMsg, target)
}

const TopicOutgoing = "TopicOutgoing"

func (dm *DefaultService) sendTarget(
	ctx context.Context, msg *message.Message, target reference.Global,
) error {

	var pulse pulse.Number
	latestPulse, err := dm.pulses.LatestTimeBeat()
	if err == nil {
		pulse = latestPulse.PulseNumber
	} else {
		// It's possible, that we try to fetch something in PM.Set()
		// In those cases, when we in the start of the system, we don't have any pulses
		// but this is not the error
		inslogger.FromContext(ctx).Warn(throw.W(err, "failed to fetch pulse"))
	}

	ctx, logger := inslogger.WithField(ctx, "sending_uuid", msg.UUID)

	msg.Metadata.Set(defaults.TraceID, inslogger.TraceID(ctx))
	sp, err := instracer.Serialize(ctx)
	if err == nil {
		msg.Metadata.Set(defaults.SpanData, string(sp))
	} else {
		logger.Error(err)
	}
	// send message and start reply goroutine
	msg.SetContext(ctx)
	_, msg, err = dm.wrapMeta(msg, target, payload.MessageHash{}, pulse)
	if err != nil {
		inslogger.FromContext(ctx).Error(throw.W(err, "failed to send message"))
		return throw.W(err, "can't wrap meta message")
	}

	logger.Debugf("sending message")
	err = dm.pub.Publish(TopicOutgoing, msg)
	if err != nil {
		return throw.W(err, "can't publish message", struct {
			Topic string
		}{
			Topic: TopicOutgoing,
		})
	}

	return nil
}

// wrapMeta wraps msg.Payload data with service fields
// and set it as byte slice back to msg.Payload.
// Note: this method has side effect - msg-argument mutating
func (dm *DefaultService) wrapMeta(
	msg *message.Message,
	receiver reference.Global,
	originHash payload.MessageHash,
	pulse pulse.Number,
) (payload.Meta, *message.Message, error) {
	msg = msg.Copy()

	payloadMeta := payload.Meta{
		Payload:    msg.Payload,
		Receiver:   receiver,
		Sender:     dm.affinity.Me(),
		Pulse:      pulse,
		OriginHash: originHash,
		ID:         []byte(msg.UUID),
	}

	buf, err := payloadMeta.Marshal()
	if err != nil {
		return payload.Meta{}, nil, throw.W(err, "wrapMeta. failed to wrap message")
	}
	msg.Payload = buf
	msg.Metadata.Set(defaults.Receiver, receiver.String())

	return payloadMeta, msg, nil
}
