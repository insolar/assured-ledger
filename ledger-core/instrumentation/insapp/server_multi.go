// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp/internal/cloud"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/network/nodeinfo"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// NetworkInitFunc should instantiate a network support for app compartment by the given configuration and root component manager.
// Returned NetworkSupport will be registered as a component and used to run app compartment.
// Returned network.Status will not be registered as a component, it can be nil, then monitoring/admin APIs will not be started.
type NetworkInitFunc = func(cert nodeinfo.Certificate) (cloud.NetworkSupport, network.Status, error)

// MultiNodeConfigFunc provides support for multi-node process initialization.
// For the given base config this handler should return a list of configurations (one per node).
type MultiNodeConfigFunc = func(baseCfg configuration.Configuration) []configuration.Configuration

func NewMulti(cfg configuration.BaseCloudConfig, multiFn MultiNodeConfigFunc) *MultiServer {
	if multiFn == nil {
		panic(throw.IllegalValue())
	}

	return &MultiServer{
		cfg:     cfg,
		multiFn: multiFn,
	}
}

type MultiServer struct {
	cfg     configuration.BaseCloudConfig
	multiFn MultiNodeConfigFunc
}

func (s *MultiServer) Serve() {
	network := cloud.NewNetwork()

	var wg sync.WaitGroup

	for _, nodeCfg := range s.multiFn(configuration.Configuration{}) {
		wg.Add(1)
		server := NewWithNetworkFn(nodeCfg, network.NetworkInitFunc)
		server.prepare()
		network.AddNode(server.certificate, server.pulseManager)
		go func() {
			defer wg.Done()
			server.run()
		}()
	}

	go func() {
		cloud.RunFakePulse(&network, s.cfg.PulsarConfiguration)
	}()

	wg.Wait()
}
