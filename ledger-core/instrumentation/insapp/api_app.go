// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp

import (
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
)

type AppComponent interface {
	GetMessageHandler() message.NoPublishHandlerFunc
	GetBeatDispatcher() beat.Dispatcher
}

type AppFactoryFunc = func(configuration.Configuration, AppComponents) AppComponent

type AppComponents struct {
	AffinityHelper affinity.Helper
	BeatHistory    beat.Accessor
	MessageSender  messagesender.Service
}
