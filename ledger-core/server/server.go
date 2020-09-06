// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package server

import (
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/server/internal/cloud"
	"github.com/insolar/assured-ledger/ledger-core/server/internal/headless"
)

type Server interface {
	Serve()
}

func NewNode(cfg configuration.Configuration) Server {
	return insapp.New(cfg, appFactory)
}

func NewMultiServer(cloudConf configuration.BaseCloudConfig) Server {
	controller := cloud.NewController()

	multiFn := func(baseCfg configuration.Configuration) ([]configuration.Configuration, insapp.NetworkInitFunc) {
		appConfigs := make([]configuration.Configuration, 0, len(cloudConf.NodeConfigPaths))
		for _, conf := range cloudConf.NodeConfigPaths {
			cfgHolder := configuration.NewHolder(conf)
			err := cfgHolder.Load()
			if err != nil {
				global.Fatal("failed to load configuration from file: ", err.Error())
			}
			appConfigs = append(appConfigs, *cfgHolder.Configuration)
		}
		return appConfigs, controller.NetworkInitFunc
	}

	return insapp.NewMulti(configuration.NewConfiguration(), appFactory, multiFn, cloud.NewPulsarWrapper(&controller, cloudConf.PulsarConfiguration))
}

func NewHeadlessNetworkNodeServer(cfg configuration.Configuration) Server {
	return insapp.New(cfg, nil, &headless.AppComponent{})
}
