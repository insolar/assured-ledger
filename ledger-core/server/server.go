// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package server

import (
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/network/pulsenetwork"
	"github.com/insolar/assured-ledger/ledger-core/server/internal/headless"
	"github.com/insolar/assured-ledger/ledger-core/testutils/cloud"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Server interface {
	Serve()
}

func NewNode(cfg configuration.Configuration) *insapp.Server {
	return insapp.New(cfg, appFactory)
}

func NewControlledMultiServer(controller cloud.Controller, configProvider insapp.ConfigurationProvider) *insapp.Server {
	multiFn := func(provider insapp.ConfigurationProvider) ([]configuration.Configuration, insapp.NetworkInitFunc) {
		conf := provider.(*CloudConfigurationProvider)
		return conf.GetAppConfigs(), controller.NetworkInitFunc
	}

	srv, _ := insapp.NewMulti(
		configProvider,
		appFactory,
		multiFn,
	)
	return srv
}

func NewMultiServer(configProvider *CloudConfigurationProvider) *insapp.Server {
	controller := cloud.NewController()
	if configProvider.GetAppConfigs == nil {
		panic("GetAppConfigs cannot be nil")
	}

	multiFn := func(provider insapp.ConfigurationProvider) ([]configuration.Configuration, insapp.NetworkInitFunc) {
		conf := provider.(*CloudConfigurationProvider)
		return conf.GetAppConfigs(), controller.NetworkInitFunc
	}

	srv, _ := insapp.NewMulti(
		configProvider,
		appFactory,
		multiFn,
		cloud.NewPulsarWrapper(&controller, configProvider.PulsarConfig, configProvider.KeyFactory),
	)
	return srv
}

func NewMultiServerWithConsensus(configProvider *CloudConfigurationProvider) *insapp.Server {
	if configProvider.GetAppConfigs == nil {
		panic("GetAppConfigs cannot be nil")
	}

	multiFn := func(provider insapp.ConfigurationProvider) ([]configuration.Configuration, insapp.NetworkInitFunc) {
		conf := provider.(*CloudConfigurationProvider)
		return conf.GetAppConfigs(), nil
	}

	pulseDistributor, err := pulsenetwork.NewDistributor(configProvider.PulsarConfig.Pulsar.PulseDistributor)
	if err != nil {
		panic(throw.W(err, "Failed to create distributor"))
	}

	srv, _ := insapp.NewMulti(
		configProvider,
		appFactory,
		multiFn,
		cloud.NewPulsarWrapper(pulseDistributor, configProvider.PulsarConfig, configProvider.KeyFactory),
	)
	return srv
}

func NewHeadlessNetworkNodeServer(cfg configuration.Configuration) *insapp.Server {
	return insapp.New(cfg, nil, &headless.AppComponent{})
}

type CloudConfigurationProvider struct {
	PulsarConfig       configuration.PulsarConfiguration
	BaseConfig         configuration.Configuration
	CertificateFactory insapp.CertManagerFactoryFunc
	KeyFactory         insapp.KeyStoreFactoryFunc
	GetAppConfigs      func() []configuration.Configuration
}

func (cp CloudConfigurationProvider) GetNamedConfig(string) configuration.Configuration {
	panic(throw.NotImplemented())
}

func (cp CloudConfigurationProvider) GetDefaultConfig() configuration.Configuration {
	return cp.BaseConfig
}

func (cp CloudConfigurationProvider) GetCertManagerFactory() insapp.CertManagerFactoryFunc {
	return cp.CertificateFactory
}

func (cp CloudConfigurationProvider) GetKeyStoreFactory() insapp.KeyStoreFactoryFunc {
	return cp.KeyFactory
}
