// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/insolar/insconfig"
)

func main() {
	cfg := readConfig()
	callbacks := NewConsensusTestCallbacks()
	netManager := NewInsolarNetManager(
		cfg.KubeParams,
		callbacks.started,
		callbacks.ready,
		callbacks.stopped,
	)

	startTest(cfg, netManager)
}

func startTest(cfg *KubeDeployToolConfig, netManager *InsolarNetManager) {
	for _, net := range cfg.NetParams {
		cfgGenerator := NewConfigGenerator(net.NodesCount)
		err := rewriteBootstrapConfigs(cfg.KubeParams, cfgGenerator)

		if err != nil {
			panic(err)
		}

		err = netManager.startNetwork(net)
		if err != nil {
			panic(err)
		}
		err = netManager.waitForReady(net)
		if err != nil {
			panic(err)
		}
		time.Sleep(net.WaitInReady)
		err = netManager.stopNetwork(net)
		if err != nil {
			panic(err)
		}
	}
}

func rewriteBootstrapConfigs(cfg KubeParams, generator *ConfigGenerator) error {
	bootstrapConfig := generator.generateBootstrapConfig()
	kustomizePatch := generator.generateKustomizePatch()
	pwConfig := generator.generatePulsewatcherConfig()

	bootstrapConfigPath := getExecutablePath() + cfg.KubeRootPath + cfg.BootstrapConfigRelPath
	err := ioutil.WriteFile(bootstrapConfigPath+"bootstrap-config.yaml", []byte(bootstrapConfig), 0644)
	if err != nil {
		return fmt.Errorf("write config failed: %w", err)
	}
	err = ioutil.WriteFile(bootstrapConfigPath+"pulsewatcher-config.yaml", []byte(pwConfig), 0644)
	if err != nil {
		return fmt.Errorf("write config failed: %w", err)
	}
	err = ioutil.WriteFile(getExecutablePath()+cfg.KubeRootPath+cfg.ManifestsRelPath+"nodes-patch.yaml", []byte(kustomizePatch), 0644)
	if err != nil {
		return fmt.Errorf("write config failed: %w", err)
	}

	return nil
}

func readConfig() *KubeDeployToolConfig {
	cfg := KubeDeployToolConfig{}
	params := insconfig.Params{
		EnvPrefix:        "kube-deploy-tool",
		ConfigPathGetter: &insconfig.DefaultPathGetter{},
	}
	insConfigurator := insconfig.New(params)
	err := insConfigurator.Load(&cfg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Config:\n%s", insConfigurator.ToYaml(cfg))
	return &cfg
}

func getExecutablePath() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Dir(ex)
}
