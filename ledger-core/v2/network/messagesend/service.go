// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/insolar/blob/master/LICENSE.md.

package messagesend

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus/meta"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/instracer"

	"github.com/pkg/errors"
)

type options struct {
	syncBody bool
}

func (o *options) applyOptions(opts ...SendOption) {
	for _, opt := range opts {
		opt(o)
	}
}

type SendOption func(*options)

func WithSyncBody() SendOption {
	return func(o *options) {
		o.syncBody = true
	}
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/network/messagesend.Service -o ./ -s _mock.go -g

type Service interface {
	// blocks if network unreachable
	SendRole(ctx context.Context, msg payload.Marshaler, role insolar.DynamicRole, object insolar.Reference, pn insolar.PulseNumber, opts ...SendOption) error
	SendTarget(ctx context.Context, msg payload.Marshaler, target insolar.Reference, opts ...SendOption) error
}

type DefaultService struct {
	pub         message.Publisher
	coordinator jet.Coordinator
	pulses      pulse.Accessor
}

func NewDefaultService(pub message.Publisher, coordinator jet.Coordinator, pulses pulse.Accessor) *DefaultService {
	return &DefaultService{
		pub:         pub,
		coordinator: coordinator,
		pulses:      pulses,
	}
}

func (dm *DefaultService) SendRole(ctx context.Context, msg payload.Marshaler, role insolar.DynamicRole, object insolar.Reference, pn insolar.PulseNumber, opts ...SendOption) error {
	waterMillMsg, err := payload.NewMessage(msg.(payload.Payload))
	if err != nil {
		return errors.Wrap(err, "Can't create watermill message")
	}

	latestPulse, err := dm.pulses.Latest(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to fetch pulse")
	}
	nodes, err := dm.coordinator.QueryRole(ctx, role, *object.GetLocal(), latestPulse.PulseNumber)
	if err != nil {
		return errors.Wrap(err, "failed to calculate role")
	}

	return dm.sendTarget(ctx, waterMillMsg, nodes[0], latestPulse.PulseNumber)
}

func (dm *DefaultService) SendTarget(ctx context.Context, msg payload.Marshaler, target insolar.Reference, opts ...SendOption) error {
	waterMillMsg, err := payload.NewMessage(msg.(payload.Payload))
	if err != nil {
		return errors.Wrap(err, "Can't create watermill message")
	}

	var pn insolar.PulseNumber
	latestPulse, err := dm.pulses.Latest(context.Background())
	if err == nil {
		pn = latestPulse.PulseNumber
	} else {
		// It's possible, that we try to fetch something in PM.Set()
		// In those cases, when we in the start of the system, we don't have any pulses
		// but this is not the error
		inslogger.FromContext(ctx).Warn(errors.Wrap(err, "failed to fetch pulse"))
	}
	return dm.sendTarget(ctx, waterMillMsg, target, pn)
}

const TopicOutgoing = "TopicOutgoing"

func (dm *DefaultService) sendTarget(
	ctx context.Context, msg *message.Message, target insolar.Reference, pulse insolar.PulseNumber,
) error {

	ctx, logger := inslogger.WithField(ctx, "sending_uuid", msg.UUID)

	msg.Metadata.Set(meta.TraceID, inslogger.TraceID(ctx))
	sp, err := instracer.Serialize(ctx)
	if err == nil {
		msg.Metadata.Set(meta.SpanData, string(sp))
	} else {
		logger.Error(err)
	}
	// send message and start reply goroutine
	msg.SetContext(ctx)
	_, msg, err = dm.wrapMeta(ctx, msg, target, payload.MessageHash{}, pulse)
	if err != nil {
		inslogger.FromContext(ctx).Error(errors.Wrap(err, "failed to send message"))
		return errors.Wrap(err, "can't wrap meta message")
	}

	logger.Debugf("sending message")
	err = dm.pub.Publish(TopicOutgoing, msg)
	if err != nil {
		return errors.Wrapf(err, "can't publish message to %s topic", TopicOutgoing)
	}

	return nil
}

// wrapMeta wraps msg.Payload data with service fields
// and set it as byte slice back to msg.Payload.
// Note: this method has side effect - msg-argument mutating
func (dm *DefaultService) wrapMeta(
	ctx context.Context,
	msg *message.Message,
	receiver insolar.Reference,
	originHash payload.MessageHash,
	pulse insolar.PulseNumber,
) (payload.Meta, *message.Message, error) {
	msg = msg.Copy()

	payloadMeta := payload.Meta{
		Polymorph:  uint32(payload.TypeMeta),
		Payload:    msg.Payload,
		Receiver:   receiver,
		Sender:     dm.coordinator.Me(),
		Pulse:      pulse,
		OriginHash: originHash,
		ID:         []byte(msg.UUID),
	}

	buf, err := payloadMeta.Marshal()
	if err != nil {
		return payload.Meta{}, nil, errors.Wrap(err, "wrapMeta. failed to wrap message")
	}
	msg.Payload = buf
	msg.Metadata.Set(meta.Receiver, receiver.String())

	return payloadMeta, msg, nil
}
