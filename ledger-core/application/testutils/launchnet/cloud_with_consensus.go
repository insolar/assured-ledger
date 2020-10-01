// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package launchnet

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/server"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func RunCloudWithConsensus(numVirtual, numLight, numHeavy int, cb func() int) int {
	fmt.Println("Run tests on cloud with consensus")

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	confProvider := prepareConfigProvider(numVirtual, numLight, numHeavy, log.DebugLevel)

	s := server.NewMultiServerWithConsensus(confProvider)

	go func() {
		sig := <-c
		fmt.Printf("Got %s signal. Aborting...\n", sig)
		s.Stop()

		os.Exit(2)
	}()

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

	err := waitForNetworkState(appConfig{Nodes: nodes}, network.CompleteNetworkState)
	if err != nil {
		fmt.Println(throw.W(err, "Can't wait for NetworkState "+network.CompleteNetworkState.String()).Error())
		return 1
	}

	code := cb()

	return code
}
