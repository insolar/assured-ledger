package launchnet

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/server"
	"github.com/insolar/assured-ledger/ledger-core/testutils/cloud"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func RunCloudWithConsensus(numVirtual, numLight, numHeavy uint, cb func() int) int {
	fmt.Println("Run tests on cloud with consensus")

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	minRoleConfiguration := cloud.NodeConfiguration{
		Virtual:       numVirtual,
		LightMaterial: numLight,
		HeavyMaterial: numHeavy,
	}

	preparedConfiguration := cloud.NodeConfiguration{
		Virtual:       numVirtual,
		LightMaterial: numLight,
		HeavyMaterial: numHeavy,
	}

	runningConfiguration := cloud.NodeConfiguration{
		Virtual:       numVirtual,
		LightMaterial: numLight,
		HeavyMaterial: numHeavy,
	}

	majorityRule := int(numVirtual + numLight + numHeavy)

	confProvider := cloud.NewConfigurationProvider(cloud.Settings{
		Prepared:     preparedConfiguration,
		Running:      runningConfiguration,
		MinRoles:     minRoleConfiguration,
		MajorityRule: majorityRule,
		Pulsar: struct {
			PulseTime int
		}{
			PulseTime: GetPulseTime(),
		},
		Log: struct {
			Level string
		}{
			Level: log.DebugLevel.String(),
		},
		CloudFileLogging: true,
	})

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
