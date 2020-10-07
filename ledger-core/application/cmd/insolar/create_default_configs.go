// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"io/ioutil"
	"path"

	"gopkg.in/yaml.v2"

	"github.com/insolar/assured-ledger/ledger-core/application/bootstrap"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
)

func writePulsarConfig(outputDir string) {
	pcfg := configuration.NewPulsarConfiguration()
	raw, err := yaml.Marshal(pcfg)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(path.Join(outputDir, "pulsar_default.yaml"), raw, 0644)
	if err != nil {
		panic(err)
	}
}

func writeBootstrapConfig(outputDir string) {
	raw, err := yaml.Marshal(bootstrap.Config{})
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(path.Join(outputDir, "bootstrap_default.yaml"), raw, 0644)
	if err != nil {
		panic(err)
	}
}

func writeNodeConfig(outputDir string) {
	cfg := configuration.NewConfiguration()
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(path.Join(outputDir, "node_default.yaml"), raw, 0644)
	if err != nil {
		panic(err)
	}
}

func writePulseWatcher(outputDir string) {
	raw, err := yaml.Marshal(configuration.NewPulseWatcherConfiguration())
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(path.Join(outputDir, "pulsewatcher_default.yaml"), raw, 0644)
	if err != nil {
		panic(err)
	}
}
