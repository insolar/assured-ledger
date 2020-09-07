// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp

import (
	"context"
	"crypto"

	"github.com/insolar/component-manager"

	"github.com/insolar/assured-ledger/ledger-core/appctl/beat"
	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/cryptography"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/mandates"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type CertManagerFactory func(publicKey crypto.PublicKey, keyProcessor cryptography.KeyProcessor, certPath string) (*mandates.CertificateManager, error)
type KeyStoreFactory func(path string) (cryptography.KeyStore, error)

type CloudConfigurationProvider struct {
	CloudConfig        configuration.BaseCloudConfig
	CertificateFactory CertManagerFactory
	KeyFactory         KeyStoreFactory
	GetAppConfigs      func() []configuration.Configuration
}

// NetworkSupport provides network-related functions to an app compartment
type NetworkSupport interface {
	beat.NodeNetwork
	nodeinfo.CertificateGetter

	CreateMessagesRouter(context.Context) messagesender.MessageRouter

	AddDispatcher(beat.Dispatcher)
	GetBeatHistory() beat.History
}

// NetworkInitFunc should instantiate a network support for app compartment by the given configuration and root component manager.
// Returned NetworkSupport will be registered as a component and used to run app compartment.
// Returned network.Status will not be registered as a component, it can be nil, then monitoring/admin APIs will not be started.
type NetworkInitFunc = func(configuration.Configuration, *component.Manager) (NetworkSupport, network.Status, error)

// MultiNodeConfigFunc provides support for multi-node process initialization.
// For the given config path and base config this handler should return a list of configurations (one per node). And NetworkInitFunc
// to initialize instantiate a network support for each app compartment (one per node). A default implementation is applied when NetworkInitFunc is nil.
type MultiNodeConfigFunc = func(baseCfg configuration.Configuration) ([]configuration.Configuration, NetworkInitFunc)

func NewMulti(cfgProvider *CloudConfigurationProvider, appFn AppFactoryFunc, multiFn MultiNodeConfigFunc, extraComponents ...interface{}) *Server {
	if multiFn == nil {
		panic(throw.IllegalValue())
	}

	conf := configuration.NewConfiguration()
	conf.Log = cfgProvider.CloudConfig.Log

	return &Server{
		cfg:          conf,
		appFn:        appFn,
		multiFn:      multiFn,
		extra:        extraComponents,
		confProvider: cfgProvider,
	}
}
