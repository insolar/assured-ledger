// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package messagesender

import (
	"context"
	"reflect"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/instracer"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type options struct {
	syncBody bool
}

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
	SendRole(ctx context.Context, msg rmsreg.GoGoSerializable, role affinity.DynamicRole, object reference.Global, pn pulse.Number, opts ...SendOption) error
	SendTarget(ctx context.Context, msg rmsreg.GoGoSerializable, target reference.Global, opts ...SendOption) error
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

func (dm *DefaultService) SendRole(ctx context.Context, msg rmsreg.GoGoSerializable, role affinity.DynamicRole, object reference.Global, pn pulse.Number, opts ...SendOption) error {
	nodes, err := dm.affinity.QueryRole(role, object, pn)
	if err != nil {
		return throw.W(err, "failed to calculate role")
	}

	return dm.sendTarget(ctx, msg, nodes[0])
}

func (dm *DefaultService) SendTarget(ctx context.Context, msg rmsreg.GoGoSerializable, target reference.Global, opts ...SendOption) error {
	return dm.sendTarget(ctx, msg, target)
}

func (dm *DefaultService) sendTarget(ctx context.Context, msg rmsreg.GoGoSerializable, target reference.Global) error {
	watermillMsgUUID := watermill.NewUUID()
	ctx, logger := inslogger.WithField(ctx, "sending_uuid", watermillMsgUUID)
	ctx, logger = inslogger.WithField(ctx, "from", dm.affinity.Me().String())
	ctx, logger = inslogger.WithField(ctx, "to", target.String())
	ctx, logger = inslogger.WithField(ctx, "type", reflect.TypeOf(msg).String()) // TODO: change that in the future

	latestPulse, err := dm.pulses.LatestTimeBeat()
	if err != nil {
		// It's possible, that we try to fetch something in PM.Set()
		// In those cases, when we in the start of the system, we don't have any pulses
		// but this is not the error
		inslogger.FromContext(ctx).Warn(throw.W(err, "failed to fetch pulse"))
	}
	latestPN := latestPulse.PulseNumber

	wrapPayload := rms.Meta{
		Sender:   rms.NewReference(dm.affinity.Me()),
		Receiver: rms.NewReference(target),
		Pulse:    latestPN,
	}
	wrapPayload.Payload.Set(msg) // TODO: here we should set message payload
	wrapPayloadBytes, err := wrapPayload.Marshal()
	if err != nil {
		inslogger.FromContext(ctx).Error(throw.W(err, "failed to send message"))
		return throw.W(err, "failed to serialize meta message")
	}

	watermillMsg := message.NewMessage(watermillMsgUUID, wrapPayloadBytes)
	watermillMsg.Metadata.Set(defaults.TraceID, inslogger.TraceID(ctx))
	watermillMsg.Metadata.Set(defaults.Receiver, target.String())

	sp, err := instracer.Serialize(ctx)
	if err == nil {
		watermillMsg.Metadata.Set(defaults.SpanData, string(sp))
	} else {
		logger.Error(err)
	}
	watermillMsg.SetContext(ctx)

	logger.Debugf("sending message")

	err = dm.pub.Publish(defaults.TopicOutgoing, watermillMsg)
	if err != nil {
		return throw.W(err, "can't publish message", struct{ Topic string }{Topic: defaults.TopicOutgoing})
	}

	return nil
}
