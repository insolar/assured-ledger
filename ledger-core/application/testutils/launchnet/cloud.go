// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package launchnet

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/server"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/cloud"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type cloudOption func(runner *CloudRunner)

type PulsarMode uint8

const (
	RegularPulsar PulsarMode = iota
	ManualPulsar
)

func WithNumVirtual(num int) func(runner *CloudRunner) {
	return func(runner *CloudRunner) {
		runner.numVirtual = num
	}
}

func WithNumLightMaterials(num int) func(runner *CloudRunner) {
	return func(runner *CloudRunner) {
		runner.numLightMaterials = num
	}
}

func WithNumHeavyMaterials(num int) func(runner *CloudRunner) {
	return func(runner *CloudRunner) {
		runner.numHeavyMaterials = num
	}
}

func WithDefaultLogLevel(level log.Level) func(runner *CloudRunner) {
	return func(runner *CloudRunner) {
		runner.defaultLogLevel = level
	}
}

func WithPulsarMode(mode PulsarMode) func(runner *CloudRunner) {
	return func(runner *CloudRunner) {
		runner.pulsarMode = mode
	}
}

func RunCloud(cb func([]string) int, options ...cloudOption) int {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ctx, abort := context.WithCancel(context.Background())

	cr := CloudRunner{
		defaultLogLevel: log.DebugLevel,
		pulsarMode:      RegularPulsar,
	}
	for _, o := range options {
		o(&cr)
	}
	cr.PrepareConfig()

	teardown, err := cr.SetupCloud(ctx)
	if err != nil {
		fmt.Println("error while setup, skip tests: ", err)
		return 1
	}
	defer teardown()
	defer abort()

	go func() {
		sig := <-c
		abort()
		fmt.Printf("Got %s signal. Aborting...\n", sig)
		teardown()

		os.Exit(2)
	}()

	pulseWatcher, config := pulseWatcherPath()

	apiAddresses := make([]string, 0, len(cr.ConfProvider.GetAppConfigs()))
	for _, el := range cr.ConfProvider.GetAppConfigs() {
		apiAddresses = append(apiAddresses, el.TestWalletAPI.Address)
	}

	code := cb(apiAddresses)

	if code != 0 {
		out, err := exec.Command(pulseWatcher, "-c", config, "-s").CombinedOutput()
		if err != nil {
			fmt.Println("PulseWatcher execution error: ", err)
			return 1
		}
		fmt.Println(string(out))
	}
	return code
}

func prepareConfigProvider(numVirtual, numLightMaterials, numHeavyMaterials int, defaultLogLevel log.Level) (*server.CloudConfigurationProvider, error) {
	cloudSettings := CloudSettings{
		Virtual: numVirtual,
		Light:   numLightMaterials,
		Heavy:   numHeavyMaterials,
		API: struct {
			TestWalletAPIPortStart int
			AdminPort              int
		}{TestWalletAPIPortStart: 32302, AdminPort: 19002},
		Log: struct{ Level string }{Level: defaultLogLevel.String()},
	}

	cloudSettings.Pulsar = struct{ PulseTime int }{PulseTime: GetPulseTime()}

	appConfigs, cloudConfig, certFactory, keyFactory := PrepareCloudConfiguration(cloudSettings)

	baseConf := configuration.Configuration{}
	baseConf.Log = cloudConfig.Log
	return &server.CloudConfigurationProvider{
		BaseConfig:         baseConf,
		PulsarConfig:       cloudConfig.PulsarConfiguration,
		CertificateFactory: certFactory,
		KeyFactory:         keyFactory,
		GetAppConfigs: func() []configuration.Configuration {
			return appConfigs
		},
	}, nil
}

type CloudRunner struct {
	numVirtual, numLightMaterials, numHeavyMaterials int

	pulsarMode PulsarMode

	defaultLogLevel log.Level

	ConfProvider *server.CloudConfigurationProvider
}

func (cr *CloudRunner) PrepareConfig() {
	var err error
	cr.ConfProvider, err = prepareConfigProvider(cr.numVirtual, cr.numLightMaterials, cr.numHeavyMaterials, cr.defaultLogLevel)
	if err != nil {
		panic(throw.W(err, "Can't prepare config provider"))
	}
}

func prepareCloudForOneShotMode(ctx context.Context, confProvider *server.CloudConfigurationProvider) server.Server {
	controller := cloud.NewController()
	s := server.NewControlledMultiServer(ctx, controller, confProvider)
	go func() {
		s.WaitStarted()

		allNodes := make(map[reference.Global]struct{})
		for _, conf := range confProvider.GetAppConfigs() {
			cert, err := confProvider.CertificateFactory(nil, nil, conf.CertificatePath)
			if err != nil {
				panic(err)
			}
			allNodes[cert.GetCertificate().GetNodeRef()] = struct{}{}
		}

		pulseGenerator := testutils.NewPulseGenerator(uint16(confProvider.PulsarConfig.Pulsar.NumberDelta), nil, nil)
		for i := 0; i < 2; i++ {
			_ = pulseGenerator.Generate()
			controller.PartialDistribute(context.Background(), pulseGenerator.GetLastPulsePacket(), allNodes)
		}
	}()

	return s
}

//nolint:goconst
func (cr CloudRunner) getPulseModeFromEnv() PulsarMode {
	pulsarOneshot := os.Getenv("PULSARD_ONESHOT")
	switch pulsarOneshot {
	case "TRUE":
		return ManualPulsar
	case "FALSE", "":
		return RegularPulsar
	default:
		panic(throw.IllegalValue())
	}
}

func (cr CloudRunner) SetupCloud(ctx context.Context) (func(), error) {
	return cr.SetupCloudCustom(ctx, cr.getPulseModeFromEnv())
}

func (cr CloudRunner) SetupCloudCustom(ctx context.Context, pulsarMode PulsarMode) (func(), error) {
	var s server.Server
	if pulsarMode == ManualPulsar {
		s = prepareCloudForOneShotMode(ctx, cr.ConfProvider)
	} else {
		s = server.NewMultiServer(ctx, cr.ConfProvider)
	}
	go func() {
		s.Serve()
	}()

	var nodes []nodeConfig
	for _, appCfg := range cr.ConfProvider.GetAppConfigs() {
		nodes = append(nodes, nodeConfig{
			AdminAPIRunner: appCfg.AdminAPIRunner,
			TestWalletAPI:  appCfg.TestWalletAPI,
		})
	}

	SetVerbose(false)
	err := waitForNetworkState(ctx, appConfig{Nodes: nodes}, network.CompleteNetworkState)
	if err != nil {
		return nil, throw.W(err, "Can't wait for NetworkState "+network.CompleteNetworkState.String())
	}
	return func() {
		s.(*insapp.Server).WaitStop()
	}, nil
}
