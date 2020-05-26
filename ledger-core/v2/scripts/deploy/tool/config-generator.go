// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package main

import (
	"fmt"
	"strconv"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/bootstrap"
	pulsewatcher "github.com/insolar/assured-ledger/ledger-core/v2/cmd/pulsewatcher/config"
)

var hostTemplate = "virtual-%d.platform-network.insolar:13831"
var certNameTemplate = "virtual-%d.json"

type ConfigGenerator struct {
	numNodes uint
}

func NewConfigGenerator(nodes uint) *ConfigGenerator {
	return &ConfigGenerator{
		numNodes: nodes,
	}
}

func (c *ConfigGenerator) generateBootstrapConfig() string {
	cfg := bootstrap.Config{
		MembersKeysDir:   "/var/data/bootstrap/configs/",
		DiscoveryKeysDir: "/var/data/bootstrap/discovery-keys/",
		KeysNameFormat:   "node-%d.json",
		MajorityRule:     5,
		MinRoles: struct {
			Virtual       uint `mapstructure:"virtual" yaml:"virtual"`
			HeavyMaterial uint `mapstructure:"heavy_material" yaml:"heavy_material"`
			LightMaterial uint `mapstructure:"light_material" yaml:"light_material"`
		}{
			Virtual:       3,
			LightMaterial: 0,
			HeavyMaterial: 0,
		},
		DiscoveryNodes: nil,
	}

	cfg.MinRoles.Virtual = c.numNodes
	cfg.MajorityRule = int(c.numNodes)

	for i := 0; i < int(c.numNodes); i++ {
		cfg.DiscoveryNodes = append(cfg.DiscoveryNodes, bootstrap.Node{
			Host:     fmt.Sprintf(hostTemplate, i),
			CertName: fmt.Sprintf(certNameTemplate, i),
		})
	}

	return c.toYaml(cfg)
}

func (c *ConfigGenerator) generateKustomizePatch() string {
	cfg := VirtualNodePatch{
		ApiVersion: "apps/v1",
		Kind:       "StatefulSet",
		Metadata: Metadata{
			Name: "virtual",
		},
		Spec: Spec{
			Selector: Selector{
				MatchLabels: MatchLabels{
					App: "virtual",
				},
			},
			Replicas: strconv.Itoa(int(c.numNodes)),
		},
	}

	return c.toYaml(cfg)
}

func (c *ConfigGenerator) generatePulsewatcherConfig() string {
	cfg := pulsewatcher.Config{
		Interval: 500 * time.Millisecond,
		Timeout:  1 * time.Second,
	}

	for i := 0; i < int(c.numNodes); i++ {
		cfg.Nodes = append(cfg.Nodes, fmt.Sprintf(hostTemplate, i))
	}

	return c.toYaml(cfg)
}

// ToYaml returns yaml marshalled struct
func (c *ConfigGenerator) toYaml(cfg interface{}) string {
	// todo clean password
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Sprintf("failed to marshal config structure: %v", err)
	}
	return string(out)
}
