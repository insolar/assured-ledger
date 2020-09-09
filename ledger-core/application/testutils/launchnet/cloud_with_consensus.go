// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build cloud_with_consensus

package launchnet

import (
	"fmt"
	"os"

	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/server"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

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
	cancelFunc := func() {
		fmt.Println("Empty cancellation function. Nothing done")
	}

	confProvider, err := prepareConfigProvider()
	if err != nil {
		return cancelFunc, throw.W(err, "Can't prepare config provider")
	}

	s := server.NewMultiServerWithConsensus(confProvider)
	go func() {
		s.Serve()
	}()

	err = launchPulsar(confProvider)
	if err != nil {
		return cancelFunc, throw.W(err, "Can't run pulsar")
	}

	var nodes []nodeConfig
	for _, appCfg := range confProvider.GetAppConfigs() {
		nodes = append(nodes, nodeConfig{
			AdminAPIRunner: appCfg.AdminAPIRunner,
			TestWalletAPI:  appCfg.TestWalletAPI,
		})
	}

	err = waitForNetworkState(appConfig{Nodes: nodes}, network.CompleteNetworkState)
	if err != nil {
		return cancelFunc, throw.W(err, "Can't wait for NetworkState "+network.CompleteNetworkState.String())
	}
	return cancelFunc, nil
}

func launchPulsar(provider *insapp.CloudConfigurationProvider) error {
	rootPath := rootPath()

	cwd, err := os.Getwd()
	if err != nil {
		return throw.W(err, "failed to get working directory")
	}
	err = os.Chdir(rootPath)
	if err != nil {
		return throw.W(err, "[ startNet  ] Can't change dir")
	}
	defer func() {
		_ = os.Chdir(cwd)
	}()

	var hosts string
	for _, conf := range provider.GetAppConfigs() {
		hosts += conf.Host.Transport.Address + ","
	}

	distrHost := "PULSARD_PULSAR_DISTR_HOST"
	os.Setenv(distrHost, hosts[:len(hosts)-1])
	defer func() {
		os.Unsetenv(distrHost)
	}()
	err = runPulsar()
	if err != nil {
		return throw.W(err, "Can't run pulsar")
	}

	return nil
}
