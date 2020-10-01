// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography/keystore"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/server"
)

func readConfig(cfgPath string) configuration.Configuration {
	cfgHolder := configuration.NewHolder(cfgPath)
	err := cfgHolder.Load()
	if err != nil {
		global.Fatal("failed to load configuration from file: ", err.Error())
	}
	return *cfgHolder.Configuration
}

func runInsolardCloud(configPath string) {
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

	baseConfig := configuration.Configuration{}
	baseConfig.Log = cloudConf.Log
	configProvider := &server.CloudConfigurationProvider{
		CertificateFactory: mandates.NewManagerReadCertificate,
		KeyFactory:         keystore.NewKeyStore,
		BaseConfig:         baseConfig,
		PulsarConfig:       cloudConf.PulsarConfiguration,
		GetAppConfigs: func() []configuration.Configuration {
			appConfigs := make([]configuration.Configuration, 0, len(cloudConf.NodeConfigPaths))
			for i, conf := range cloudConf.NodeConfigPaths {
				cfgHolder := configuration.NewHolder(conf)
				err := cfgHolder.Load()
				if err != nil {
					global.Fatal("failed to load configuration from file: ", err.Error())
				}
				fmt.Printf("Starts with configuration [%d/%d]:\n%s\n", i+1, len(cloudConf.NodeConfigPaths), configuration.ToString(&cfgHolder.Configuration))
				appConfigs = append(appConfigs, *cfgHolder.Configuration)
			}
			return appConfigs
		},
	}

	s := server.NewMultiServer(configProvider)

	s.Serve()
}
