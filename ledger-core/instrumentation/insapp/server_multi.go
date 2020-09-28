// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp

import (
	"context"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// NetworkSupport provides network-related functions to an app compartment
type NetworkSupport interface {
	beat.NodeNetwork
	nodeinfo.CertificateGetter

	CreateMessagesRouter(context.Context) messagesender.MessageRouter

	AddDispatcher(beat.Dispatcher)
	GetBeatHistory() beat.History
}

// NetworkInitFunc should instantiate a network support for app compartment by the given configuration and root component manager.
// Returned NetworkSupport will be registered as a component and used to run app compartment.
// Returned network.Status will not be registered as a component, it can be nil, then monitoring/admin APIs will not be started.
type NetworkInitFunc = func(configuration.Configuration, *component.Manager) (NetworkSupport, network.Status, error)

// MultiNodeConfigFunc provides support for multi-node process initialization.
// For the given config path and base config this handler should return a list of configurations (one per node). And NetworkInitFunc
// to initialize instantiate a network support for each app compartment (one per node). A default implementation is applied when NetworkInitFunc is nil.
type MultiNodeConfigFunc = func(baseCfg ConfigurationProvider) ([]configuration.Configuration, NetworkInitFunc)

func NewMulti(cfgProvider ConfigurationProvider, appFn AppFactoryFunc, multiFn MultiNodeConfigFunc, extraComponents ...interface{}) *Server {
	if multiFn == nil {
		panic(throw.IllegalValue())
	}

	return &Server{
		appFn:        appFn,
		multiFn:      multiFn,
		extra:        extraComponents,
		confProvider: cfgProvider,
		started:      make(chan struct{}),
	}
}
