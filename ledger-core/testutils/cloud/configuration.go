// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package cloud

import (
	"crypto"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/network/consensus/gcpv2/api/member"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func NewConfigurationProvider(cloudSettings Settings) *ConfigurationProvider {
	appConfigs, cloudConfig, certs, nodes := prepareCloudConfiguration(cloudSettings)

	availableNodesByRole := make(map[member.PrimaryRole][]reference.Global, 3)
	for _, node := range nodes {
		availableNodesByRole[node.role] = append(availableNodesByRole[node.role], node.reference)
	}

	baseConf := configuration.Configuration{}
	baseConf.Log = cloudConfig.Log
	res := &ConfigurationProvider{
		BaseConfig:           baseConf,
		PulsarConfig:         cloudConfig.PulsarConfiguration,
		CertificateFactory:   makeCertManagerFactory(certs),
		KeyFactory:           makeKeyFactory(append(nodes, preparePulsarKeys(cloudConfig))),
		nodeConfigs:          appConfigs,
		availableNodesByRole: availableNodesByRole,
		runningNodesByRole:   make(map[member.PrimaryRole][]reference.Global, 3),
	}

	res.runningNodesSetup(cloudSettings.Running)

	return res
}

func (cp *ConfigurationProvider) GetAppConfigs() []configuration.Configuration {
	if cp.FixedConfigs != nil {
		return cp.FixedConfigs
	}

	appConfigs := make([]configuration.Configuration, 0, cp.runningNodeCount)
	for _, nodes := range cp.runningNodesByRole {
		for _, nodeRef := range nodes {
			appConfigs = append(appConfigs, cp.nodeConfigs[nodeRef])
		}
	}
	return appConfigs
}

func (cp *ConfigurationProvider) RunNode(role member.PrimaryRole) reference.Global {
	availableNodes := cp.availableNodesByRole[role]
	if len(availableNodes) == 0 {
		panic(throw.IllegalState())
	}
	var nodeRef reference.Global
	nodeRef, cp.availableNodesByRole[role] = availableNodes[0], availableNodes[1:]
	cp.runningNodesByRole[role] = append(cp.runningNodesByRole[role], nodeRef)
	return nodeRef
}

func (cp *ConfigurationProvider) FreeNode(role member.PrimaryRole, nodeRef reference.Global) {
	reservedNodes := cp.runningNodesByRole[role]
	if len(reservedNodes) == 0 {
		panic(throw.IllegalState())
	}

	pos := -1
	for i := range reservedNodes {
		if reservedNodes[i].Equal(nodeRef) {
			pos = i
			break
		}
	}

	if pos < 0 {
		panic(throw.IllegalValue())
	}

	availableNodes := cp.availableNodesByRole[role]
	for i := range availableNodes {
		if availableNodes[i] == nodeRef {
			panic(throw.IllegalValue())
		}
	}

	cp.runningNodesByRole[role] = append(reservedNodes[:pos], reservedNodes[pos+1:]...)
	cp.availableNodesByRole[role] = append(availableNodes, nodeRef)
}

func (cp *ConfigurationProvider) runningNodesSetup(cfg NodeConfiguration) {
	for cfg.Virtual > 0 {
		cp.RunNode(member.PrimaryRoleVirtual)
		cfg.Virtual--
	}

	for cfg.LightMaterial > 0 {
		cp.RunNode(member.PrimaryRoleLightMaterial)
		cfg.LightMaterial--
	}

	for cfg.HeavyMaterial > 0 {
		cp.RunNode(member.PrimaryRoleHeavyMaterial)
		cfg.HeavyMaterial--
	}
}

func (cp *ConfigurationProvider) GetRunningNodesMap() map[reference.Global]configuration.Configuration {
	appConfigs := make(map[reference.Global]configuration.Configuration, cp.runningNodeCount)
	for _, nodes := range cp.runningNodesByRole {
		for _, nodeRef := range nodes {
			appConfigs[nodeRef] = cp.nodeConfigs[nodeRef]
		}
	}
	return appConfigs
}

type ConfigurationProvider struct {
	runningNodeCount int

	cloudFileLogging bool

	runningNodesByRole   map[member.PrimaryRole][]reference.Global
	availableNodesByRole map[member.PrimaryRole][]reference.Global

	nodeConfigs map[reference.Global]configuration.Configuration

	PulsarConfig       configuration.PulsarConfiguration
	BaseConfig         configuration.Configuration
	CertificateFactory insapp.CertManagerFactoryFunc
	KeyFactory         insapp.KeyStoreFactoryFunc

	FixedConfigs []configuration.Configuration
}

func (cp *ConfigurationProvider) SetFileLogging() {
	cp.cloudFileLogging = true
}

func (cp *ConfigurationProvider) GetNamedConfig(refStr string) configuration.Configuration {
	nodeRef, err := reference.GlobalFromString(refStr)
	if err != nil {
		panic(throw.IllegalValue())
	}

	conf, found := cp.nodeConfigs[nodeRef]
	if !found {
		panic(throw.IllegalState())
	}
	return conf
}

func (cp ConfigurationProvider) GetDefaultConfig() configuration.Configuration {
	return cp.BaseConfig
}

func (cp ConfigurationProvider) GetCertManagerFactory() insapp.CertManagerFactoryFunc {
	return cp.CertificateFactory
}

func (cp ConfigurationProvider) GetKeyStoreFactory() insapp.KeyStoreFactoryFunc {
	return cp.KeyFactory
}

type InMemoryKeyStore struct {
	Key crypto.PrivateKey
}

func (ks InMemoryKeyStore) GetPrivateKey(string) (crypto.PrivateKey, error) {
	return ks.Key, nil
}

type NodeConfiguration struct {
	Virtual       uint
	HeavyMaterial uint
	LightMaterial uint
}

func (c NodeConfiguration) IsZero() bool {
	return c.Virtual == 0 && c.LightMaterial == 0 && c.HeavyMaterial == 0
}

type Settings struct {
	Prepared NodeConfiguration
	Running  NodeConfiguration
	MinRoles NodeConfiguration

	MajorityRule int

	Pulsar struct {
		PulseTime int
	}
	Log struct {
		Level string
	}
	CloudFileLogging bool
}
