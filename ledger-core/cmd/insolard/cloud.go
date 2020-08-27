// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"io/ioutil"

	jww "github.com/spf13/jwalterweatherman"
	"gopkg.in/yaml.v2"

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

func runInsolardCloud(configPath string) {
	jww.SetStdoutThreshold(jww.LevelDebug)

	if err := psAgentLauncher(); err != nil {
		global.Fatal("Failed to launch gops agent: %s", err)
	}

	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		global.Fatal("failed ro read config file", err)
	}

	var cloudConf configuration.BaseCloudConfig
	err = yaml.Unmarshal(yamlFile, &cloudConf)
	if err != nil {
		global.Fatal("Failed to parse YAML file", err)
	}

	var multiFn insapp.MultiNodeConfigFunc
	multiFn = func(cfgPath string, baseCfg configuration.Configuration) ([]configuration.Configuration, insapp.NetworkInitFunc) {
		appConfigs := []configuration.Configuration{}
		for _, conf := range cloudConf.NodeConfigPaths {
			appConfigs = append(appConfigs, readConfig(conf))
		}
		return appConfigs, nil
	}

	s := server.NewMultiServer(configPath, multiFn)

	s.Serve()
}
