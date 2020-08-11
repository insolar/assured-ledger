// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/crypto"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/injector"
)

type AppComponent interface {
	// Init(ctx context.Context) error
	// Start(ctx context.Context) error
	// Stop(ctx context.Context) error

	GetMessageHandler() message.NoPublishHandlerFunc
	GetBeatDispatcher() beat.Dispatcher
}

type AppFactoryFunc = func(context.Context, configuration.Configuration, AppComponents) (AppComponent, error)

type AppComponents struct {
	AffinityHelper affinity.Helper
	BeatHistory    beat.History
	MessageSender  messagesender.Service
	CryptoScheme   crypto.PlatformScheme
}

func (v AppComponents) AddInterfaceDependencies(container injector.DependencyContainer) {
	if v.AffinityHelper != nil {
		injector.AddInterfaceDependency(container, &v.AffinityHelper)
	}
	if v.BeatHistory != nil {
		injector.AddInterfaceDependency(container, &v.BeatHistory)
	}
	if v.MessageSender != nil {
		injector.AddInterfaceDependency(container, &v.MessageSender)
	}
	if v.CryptoScheme != nil {
		injector.AddInterfaceDependency(container, &v.CryptoScheme)
	}
}
