// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package watermill

import (
	"context"
	"io"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/insolar/defaults"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/logwatermill"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ messagesender.MessageRouter = Router{}

func NewRouter(ctx context.Context, outHandler message.NoPublishHandlerFunc) Router {
	if outHandler == nil {
		panic(throw.IllegalValue())
	}

	wmLogger := logwatermill.NewWatermillLogAdapter(inslogger.FromContext(ctx))
	pubsub := gochannel.NewGoChannel(gochannel.Config{}, wmLogger)
	return Router{
		ctx:        ctx,
		logger:     wmLogger,
		Pub:        pubsub,
		sub:        pubsub,
		outHandler: outHandler,
	}
}

type Router struct {
	ctx        context.Context
	logger     *logwatermill.WatermillLogAdapter
	Pub        message.Publisher
	sub        message.Subscriber
	outHandler message.NoPublishHandlerFunc
}

func (v Router) IsZero() bool {
	return v.Pub == nil
}

func (v Router) CreateMessageSender(helper affinity.Helper, accessor beat.History) messagesender.Service {
	return messagesender.NewDefaultService(v.Pub, helper, accessor)
}

func (v Router) SubscribeForMessages(inHandler func(beat.Message) error) (stopFn func()) {
	switch {
	case v.sub == nil:
		panic(throw.IllegalState())
	case v.outHandler == nil:
		panic(throw.IllegalState())
	case inHandler == nil:
		panic(throw.IllegalState())
	}

	inRouter, err := message.NewRouter(message.RouterConfig{}, v.logger)
	if err != nil {
		panic(err)
	}
	outRouter, err := message.NewRouter(message.RouterConfig{}, v.logger)
	if err != nil {
		panic(err)
	}

	outRouter.AddNoPublisherHandler(
		"OutgoingHandler",
		defaults.TopicOutgoing,
		v.sub,
		v.outHandler,
	)

	inRouter.AddNoPublisherHandler(
		"IncomingHandler",
		defaults.TopicIncoming,
		v.sub,
		func(msg *message.Message) error {
			bm := beat.NewMessageExt(msg.UUID, msg.Payload, msg)
			bm.Metadata = msg.Metadata
			return inHandler(bm)
		},
	)

	startRouter(v.ctx, inRouter)
	startRouter(v.ctx, outRouter)

	return stopWatermill(v.ctx, inRouter, outRouter)
}

func stopWatermill(ctx context.Context, routers ...io.Closer) func() {
	return func() {
		for _, r := range routers {
			if err := r.Close(); err != nil {
				inslogger.FromContext(ctx).Error("Error while stopping router", err)
			}
		}
	}
}

func startRouter(ctx context.Context, router *message.Router) {
	go func() {
		if err := router.Run(ctx); err != nil {
			inslogger.FromContext(ctx).Error("Error while running router", err)
		}
	}()
	<-router.Running()
}
