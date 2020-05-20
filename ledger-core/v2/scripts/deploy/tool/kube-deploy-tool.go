// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"fmt"
	"os/exec"

	"github.com/insolar/insconfig"
)

func main() {
	cfg := readConfig()

	runNetwork(cfg)
}

func runNetwork(cfg *KubeDeployTool) {
	out, err := exec.Command("date").Output()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("The date is %s\n", out)
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
