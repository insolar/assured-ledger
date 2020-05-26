// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/insolar/insconfig"
)

const Kubectl = "kubectl"

func main() {
	cfg := readConfig()
	cfgGenerator := NewConfigGenerator(cfg.NodesCount)
	netManager := NewInsolarNetManager()
	generateBootstrapConfigs(cfg, cfgGenerator)
	os.Exit(0)

	err := netManager.startNetwork(cfg)
	if err != nil {
		panic(err)
	}
	callCreated()
	err = netManager.waitForReady()
	if err != nil {
		panic(err)
	}
	callReady()
	err = netManager.stopNetwork(cfg)
	if err != nil {
		panic(err)
	}
	callStopped()
}

func generateBootstrapConfigs(cfg *KubeDeployToolConfig, generator *ConfigGenerator) {
	bootstrapConfig := generator.generateBootstrapConfig()
	kustomizePatch := generator.generateKustomizePatch()
	pwConfig := generator.generatePulsewatcherConfig()

	fmt.Println("debug")
	fmt.Println(bootstrapConfig)
	fmt.Println(kustomizePatch)
	fmt.Println(pwConfig)

	bootstrapConfigPath := getExecutablePath() + cfg.ManifestsPath + cfg.BootstrapConfigRelPath
	ioutil.WriteFile(bootstrapConfigPath+"bootstrap-config.yaml", []byte(bootstrapConfig), 0644)
	ioutil.WriteFile(bootstrapConfigPath+"pulsewatcher-config.yaml", []byte(pwConfig), 0644)
	ioutil.WriteFile(getExecutablePath()+cfg.ManifestsPath+"nodes-patch.yaml", []byte(kustomizePatch), 0644)
}

func callStopped() {

}

func callReady() {

}

func callCreated() {

}

type InsolarNetManager struct {
	// todo add callbacks and cfg and interface
}

func NewInsolarNetManager() *InsolarNetManager {
	return &InsolarNetManager{}
}

func (i *InsolarNetManager) startNetwork(cfg *KubeDeployToolConfig) error {
	out, _ := exec.Command("pwd").CombinedOutput()
	fmt.Printf("path: %s\n", out)

	out, err := exec.Command(Kubectl, "apply", "-k", getExecutablePath()+cfg.ManifestsPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("run failed: %s %w", string(out), err)
	}
	fmt.Printf("Output: %s\n", out)
	return nil
}

func (i *InsolarNetManager) waitForReady() error {
	for i := 0; i < 60; i++ {
		args := []string{
			"-n",
			"insolar",
			"get",
			"po",
			"bootstrap",
			"-o",
			"jsonpath=\"{.status.phase}\"",
		}
		cmd := exec.Command(Kubectl, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("bootstrap check failed: %s %w", string(out), err)
		}
		if string(out) == "\"Succeeded\"" {
			fmt.Println("bootstrap finished")
			break
		}
		time.Sleep(1 * time.Second)
	}

	for i := 0; i < 60; i++ {
		args := []string{
			"-n",
			"insolar",
			"exec",
			"-i",
			"deploy/pulsewatcher",
			"--",
			`pulsewatcher`,
			"-c",
			"/etc/pulsewatcher/pulsewatcher.yaml",
			`-s`,
		}
		cmd := exec.Command(Kubectl, args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("insolar ready check failed: %s %w", string(out), err)
		}
		if strings.Contains(string(out), "READY") && !strings.Contains(string(out), "NOT") {
			fmt.Println("insolar has been started")
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	return errors.New("insolar has not been started")
}

func (i *InsolarNetManager) stopNetwork(cfg *KubeDeployToolConfig) error {
	out, err := exec.Command(Kubectl, "delete", "-k", getExecutablePath()+cfg.ManifestsPath).CombinedOutput()
	if err != nil {
		fmt.Errorf("stop failed: %s %w", string(out), err)
	}
	fmt.Printf("Output: %s\n", out)
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
