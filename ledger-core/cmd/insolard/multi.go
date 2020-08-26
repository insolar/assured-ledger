// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	jww "github.com/spf13/jwalterweatherman"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/server"
)

func readConfig(cfgPath string) configuration.Configuration {
	cfgHolder := configuration.NewHolder(cfgPath)
	err := cfgHolder.Load()
	if err != nil {
		global.Warn("failed to load configuration from file: ", err.Error())
	}
	return *cfgHolder.Configuration
}

// nolint:unparam
func runInsolardCloud(configPath string) {
	jww.SetStdoutThreshold(jww.LevelDebug)

	if err := psAgentLauncher(); err != nil {
		global.Warnf("Failed to launch gops agent: %s", err)
	}

	var multiFn insapp.MultiNodeConfigFunc
	multiFn = func(cfgPath string, baseCfg configuration.Configuration) ([]configuration.Configuration, insapp.NetworkInitFunc) {
		appConfigs := []configuration.Configuration{}
		for _, conf := range []string{
			".artifacts/launchnet/discoverynodes/1/insolard.yaml",
			".artifacts/launchnet/discoverynodes/2/insolard.yaml",
			".artifacts/launchnet/discoverynodes/3/insolard.yaml",
			".artifacts/launchnet/discoverynodes/4/insolard.yaml",
			".artifacts/launchnet/discoverynodes/5/insolard.yaml",
		} {
			appConfigs = append(appConfigs, readConfig(conf))
		}
		return appConfigs, nil
	}

	s := server.NewVirtualMultiServer(configPath, multiFn)

	s.Serve()
}
