// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package server

import (
	"context"

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

func NewNode(ctx context.Context, cfg configuration.Configuration) Server {
	return insapp.New(ctx, cfg, appFactory)
}

func NewControlledMultiServer(ctx context.Context, controller cloud.Controller, configProvider insapp.ConfigurationProvider) *insapp.Server {
	multiFn := func(provider insapp.ConfigurationProvider) ([]configuration.Configuration, insapp.NetworkInitFunc) {
		conf := provider.(*CloudConfigurationProvider)
		return conf.GetAppConfigs(), controller.NetworkInitFunc
	}

	return insapp.NewMulti(
		ctx,
		configProvider,
		appFactory,
		multiFn,
	)
}

func NewMultiServer(ctx context.Context, configProvider *CloudConfigurationProvider) Server {
	controller := cloud.NewController()
	if configProvider.GetAppConfigs == nil {
		panic("GetAppConfigs cannot be nil")
	}

	multiFn := func(provider insapp.ConfigurationProvider) ([]configuration.Configuration, insapp.NetworkInitFunc) {
		conf := provider.(*CloudConfigurationProvider)
		return conf.GetAppConfigs(), controller.NetworkInitFunc
	}

	return insapp.NewMulti(
		ctx,
		configProvider,
		appFactory,
		multiFn,
		cloud.NewPulsarWrapper(&controller, configProvider.PulsarConfig, configProvider.KeyFactory),
	)
}

func NewMultiServerWithConsensus(ctx context.Context, configProvider *CloudConfigurationProvider) Server { // nolint:interfacer
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

	return insapp.NewMulti(
		ctx,
		configProvider,
		appFactory,
		multiFn,
		cloud.NewPulsarWrapper(pulseDistributor, configProvider.PulsarConfig, configProvider.KeyFactory),
	)
}

func NewHeadlessNetworkNodeServer(ctx context.Context, cfg configuration.Configuration) Server {
	return insapp.New(ctx, cfg, nil, &headless.AppComponent{})
}

type CloudConfigurationProvider struct {
	PulsarConfig       configuration.PulsarConfiguration
	BaseConfig         configuration.Configuration
	CertificateFactory insapp.CertManagerFactory
	KeyFactory         insapp.KeyStoreFactory
	GetAppConfigs      func() []configuration.Configuration
}

func (cp CloudConfigurationProvider) Config() configuration.Configuration {
	return cp.BaseConfig
}

func (cp CloudConfigurationProvider) GetCertManagerFactory() insapp.CertManagerFactory {
	return cp.CertificateFactory
}

func (cp CloudConfigurationProvider) GetKeyStoreFactory() insapp.KeyStoreFactory {
	return cp.KeyFactory
}
