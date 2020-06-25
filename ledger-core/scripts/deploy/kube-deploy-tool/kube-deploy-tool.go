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

	"github.com/insolar/insconfig"
)

func main() {
	os.Exit(runSuite())
}

func runSuite() int {
	cfg := readConfig()
	callbacks := NewConsensusTestCallbacks()
	insolarManager := NewInsolarNetManager(
		cfg.KubeParams,
		callbacks.started,
		callbacks.ready,
		callbacks.stopped,
	)

	if cfg.KubeParams.Prometheus.Enabled {
		promManager := NewPrometheusManager(cfg.KubeParams)
		err := promManager.start()
		if err != nil {
			panic(err)
		}
		defer func() {
			err := promManager.stop()
			if err != nil {
				fmt.Print(err.Error())
			}
		}()
	}
	err := insolarManager.checkDependencies()
	if err != nil {
		panic(err)
	}

	startTest(cfg, insolarManager)

	err = callbacks.suiteFinished(cfg)
	if err != nil {
		fmt.Println("Error: ", err)
		return 1
	}
	return 0
}

func startTest(cfg *KubeDeployToolConfig, insolarManager *InsolarNetManager) {
	if cfg.KubeParams.LogCollector.Enabled {
		err := insolarManager.cleanLogDir()
		if err != nil {
			panic(err)
		}
	}

	for _, net := range cfg.NetParams {
		cfgGenerator := NewConfigGenerator(net.NodesCount)
		err := rewriteBootstrapConfigs(cfg.KubeParams, cfgGenerator)
		if err != nil {
			panic(err)
		}

		startAndReady := func(net NetParams) error {
			err := insolarManager.start(net)
			if err != nil {
				return fmt.Errorf("failed to start insolar: %w\n nodes count: %d", err, net.NodesCount)
			}

			err = insolarManager.waitForReady(net)
			if err != nil {
				return fmt.Errorf("nodes count: %d, failed to wait insolar ready: %w", net.NodesCount, err)
			}
			return nil
		}

		startAndReadyError := startAndReady(net)
		if startAndReadyError == nil {
			err := insolarManager.waitInReady(net)
			if err != nil {
				fmt.Printf("failed to waitInReady: %s\n nodes count: %d", err.Error(), net.NodesCount)
			}
		}

		if cfg.KubeParams.LogCollector.Enabled {
			err := insolarManager.collectLogs(net)
			if err != nil {
				fmt.Printf("failed to collect logs: %s\n nodes count: %d", err.Error(), net.NodesCount)
			}
		}

		if startAndReadyError != nil {
			fmt.Print(startAndReadyError.Error())
			if cfg.KubeParams.LeaveInsolarAliveIfFailed {
				panic("LeaveInsolarAliveIfFailed is true, stopping deploy tool")
			}
		}

		err = insolarManager.stop(net)
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
	nodesPatchPath := getExecutablePath() + cfg.KubeRootPath + cfg.ManifestsRelPath
	err := ioutil.WriteFile(bootstrapConfigPath+"bootstrap-config.yaml", []byte(bootstrapConfig), 0644)
	if err != nil {
		return fmt.Errorf("write config failed: %w", err)
	}
	err = ioutil.WriteFile(bootstrapConfigPath+"pulsewatcher-config.yaml", []byte(pwConfig), 0644)
	if err != nil {
		return fmt.Errorf("write config failed: %w", err)
	}
	err = ioutil.WriteFile(nodesPatchPath+"nodes-patch.yaml", []byte(kustomizePatch), 0644)
	if err != nil {
		return fmt.Errorf("write config failed: %w", err)
	}

	return nil
}

func readConfig() *KubeDeployToolConfig {
	cfg := KubeDeployToolConfig{}
	params := insconfig.Params{
		EnvPrefix:        "kubedeploytool",
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
