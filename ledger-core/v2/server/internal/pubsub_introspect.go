// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build introspection

package internal

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/insolar/component-manager"
	"golang.org/x/net/context"

	"github.com/insolar/assured-ledger/ledger-core/v2/configuration"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/introspector"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/introspector/pubsubwrap"
)

// PublisherWrapper setups and returns introspection wrapper for message.Publisher.
func PublisherWrapper(
	ctx context.Context,
	cm *component.Manager,
	cfg configuration.Introspection,
	pb message.Publisher,
) message.Publisher {
	pw := pubsubwrap.NewPublisherWrapper(pb)

	// init pubsub middlewares and add them to wrapper
	mStat := pubsubwrap.NewMessageStatByType()
	mLocker := pubsubwrap.NewMessageLockerByType(ctx)
	pw.Middleware(mStat)
	pw.Middleware(mLocker)

	// create introspection server with service which implements introproto.PublisherServer
	service := pubsubwrap.NewPublisherService(mLocker, mStat)
	iSrv := introspector.NewServer(cfg.Addr, service)

	// use component manager for lifecycle (component.Manager calls Start/Stop on server instance)
	cm.Register(iSrv)

	return pw
}
