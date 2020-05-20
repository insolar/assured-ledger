// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"errors"
	"fmt"
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

	err := runNetwork(cfg)
	if err != nil {
		panic(err)
	}
	err = waitForReady()
	if err != nil {
		panic(err)
	}
	err = stopNetwork(cfg)
	if err != nil {
		panic(err)
	}
}

func waitForReady() error {
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

func runNetwork(cfg *KubeDeployTool) error {
	// fmt.Println(command)
	out, _ := exec.Command("pwd").CombinedOutput()
	fmt.Printf("path: %s\n", out)

	out, err := exec.Command(Kubectl, "apply", "-k", getExecutablePath()+cfg.ManifestsPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("run failed: %s %w", string(out), err)
	}
	fmt.Printf("Output: %s\n", out)
	return nil
}

func stopNetwork(cfg *KubeDeployTool) error {
	out, err := exec.Command(Kubectl, "delete", "-k", getExecutablePath()+cfg.ManifestsPath).CombinedOutput()
	if err != nil {
		fmt.Errorf("stop failed: %s %w", string(out), err)
	}
	fmt.Printf("Output: %s\n", out)
	return nil
}

func readConfig() *KubeDeployTool {
	cfg := KubeDeployTool{}
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
