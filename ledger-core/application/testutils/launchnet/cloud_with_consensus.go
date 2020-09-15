// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build cloud_with_consensus

package launchnet

import (
	"fmt"

	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/server"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func isCloudMode() bool {
	return true
}

func Run(cb func() int) int {
	fmt.Println("Run tests on cloud with consensus")

	cancelFunc, err := setupCloudWithConsensus()
	if err != nil {
		fmt.Println("error while setup cloud with consensus, skip tests: ", err)
		return 1
	}

	code := cb()
	cancelFunc()

	return code
}

func setupCloudWithConsensus() (func(), error) {
	cancelFunc := func() {}

	confProvider, err := prepareConfigProvider()
	if err != nil {
		return cancelFunc, throw.W(err, "Can't prepare config provider")
	}

	s := server.NewMultiServerWithConsensus(confProvider)
	go func() {
		s.Serve()
	}()

	var nodes []nodeConfig
	for _, appCfg := range confProvider.GetAppConfigs() {
		nodes = append(nodes, nodeConfig{
			AdminAPIRunner: appCfg.AdminAPIRunner,
			TestWalletAPI:  appCfg.TestWalletAPI,
		})
	}

	err = waitForNetworkState(appConfig{Nodes: nodes}, network.CompleteNetworkState, true)
	if err != nil {
		return cancelFunc, throw.W(err, "Can't wait for NetworkState "+network.CompleteNetworkState.String())
	}
	return cancelFunc, nil
}
