// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package server

import (
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/server/internal/cloud"
	"github.com/insolar/assured-ledger/ledger-core/server/internal/headless"
)

type Server interface {
	Serve()
}

func NewNode(cfg configuration.Configuration) Server {
	return insapp.New(cfg, appFactory)
}

func NewMultiServer(configProvider insapp.CloudConfigurationProvider) Server {
	controller := cloud.NewController()
	if configProvider.GetAppConfigs == nil {
		panic("GetAppConfigs cannot be nil")
	}
	cloudConf := configProvider.CloudConfig

	multiFn := func(baseCfg configuration.Configuration) ([]configuration.Configuration, insapp.NetworkInitFunc) {
		return configProvider.GetAppConfigs(), controller.NetworkInitFunc
	}

	return insapp.NewMulti(
		configProvider,
		appFactory,
		multiFn,
		cloud.NewPulsarWrapper(&controller, cloudConf.PulsarConfiguration, configProvider.KeyFactory),
	)
}

func NewHeadlessNetworkNodeServer(cfg configuration.Configuration) Server {
	return insapp.New(cfg, nil, &headless.AppComponent{})
}
