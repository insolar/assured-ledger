// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package server

import (
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/network/pulsenetwork"
	"github.com/insolar/assured-ledger/ledger-core/server/internal/cloud"
	"github.com/insolar/assured-ledger/ledger-core/server/internal/headless"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Server interface {
	Serve()
}

func NewNode(cfg configuration.Configuration) Server {
	return insapp.New(cfg, appFactory)
}

func NewMultiServer(configProvider *insapp.CloudConfigurationProvider) Server {
	controller := cloud.NewController()
	if configProvider.GetAppConfigs == nil {
		panic("GetAppConfigs cannot be nil")
	}

	multiFn := func(provider insapp.ConfigurationProvider) ([]configuration.Configuration, insapp.NetworkInitFunc) {
		conf := provider.(*insapp.CloudConfigurationProvider)
		return conf.GetAppConfigs(), controller.NetworkInitFunc
	}

	return insapp.NewMulti(
		configProvider,
		appFactory,
		multiFn,
		cloud.NewPulsarWrapper(&controller, configProvider.PulsarConfig, configProvider.KeyFactory),
	)
}

func NewMultiServerWithConsensus(configProvider *insapp.CloudConfigurationProvider) Server { // nolint:interfacer
	if configProvider.GetAppConfigs == nil {
		panic("GetAppConfigs cannot be nil")
	}

	multiFn := func(provider insapp.ConfigurationProvider) ([]configuration.Configuration, insapp.NetworkInitFunc) {
		conf := provider.(*insapp.CloudConfigurationProvider)
		return conf.GetAppConfigs(), nil
	}

	pulseDistributor, err := pulsenetwork.NewDistributor(configProvider.PulsarConfig.Pulsar.PulseDistributor)
	if err != nil {
		panic(throw.W(err, "Failed to create distributor"))
	}

	return insapp.NewMulti(
		configProvider,
		appFactory,
		multiFn,
		cloud.NewPulsarWrapper(pulseDistributor, configProvider.PulsarConfig, configProvider.KeyFactory),
	)
}

func NewHeadlessNetworkNodeServer(cfg configuration.Configuration) Server {
	return insapp.New(cfg, nil, &headless.AppComponent{})
}
