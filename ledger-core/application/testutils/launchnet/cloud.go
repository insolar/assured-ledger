// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package launchnet

import (
	"context"
	"os"
	"strconv"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/network"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/server"
	"github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/cloud"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var (
	numVirtual        = 5
	numLightMaterials = 0
	numHeavyMaterials = 0
)

type PulsarMode uint8

const (
	RegularPulsar PulsarMode = iota
	ManualPulsar
)

func prepareConfigProvider() (*server.CloudConfigurationProvider, error) {
	pulseEnv := os.Getenv("PULSARD_PULSAR_PULSETIME")
	var pulseTime int
	var err error
	if len(pulseEnv) != 0 {
		pulseTime, err = strconv.Atoi(pulseEnv)
		if err != nil {
			return nil, throw.W(err, "Can't convert env var")
		}
	}

	cloudSettings := CloudSettings{
		Virtual: numVirtual,
		Light:   numLightMaterials,
		Heavy:   numHeavyMaterials,
		API: struct {
			TestWalletAPIPortStart int
			AdminPort              int
		}{TestWalletAPIPortStart: 32302, AdminPort: 19002},
	}

	if pulseTime != 0 {
		cloudSettings.Pulsar = struct{ PulseTime int }{PulseTime: pulseTime}
	}

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
	ConfProvider *server.CloudConfigurationProvider
}

func (cr CloudRunner) SetNumVirtuals(n int) {
	numVirtual = n
}

func (cr *CloudRunner) PrepareConfig() {
	var err error
	cr.ConfProvider, err = prepareConfigProvider()
	if err != nil {
		panic(throw.W(err, "Can't prepare config provider"))
	}
}

func prepareCloudForOneShotMode(confProvider *server.CloudConfigurationProvider) server.Server {
	controller := cloud.NewController()
	s := server.NewControlledMultiServer(controller, confProvider)
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

func (cr CloudRunner) SetupCloud() (func(), error) {
	return cr.SetupCloudCustom(RegularPulsar)
}

func (cr CloudRunner) SetupCloudCustom(pulsarMode PulsarMode) (func(), error) {
	var s server.Server
	if pulsarMode == ManualPulsar {
		s = prepareCloudForOneShotMode(cr.ConfProvider)
	} else {
		s = server.NewMultiServer(cr.ConfProvider)
	}
	go func() {
		s.Serve()
	}()

	cancelFunc := s.(*insapp.Server).Stop

	var nodes []nodeConfig
	for _, appCfg := range cr.ConfProvider.GetAppConfigs() {
		nodes = append(nodes, nodeConfig{
			AdminAPIRunner: appCfg.AdminAPIRunner,
			TestWalletAPI:  appCfg.TestWalletAPI,
		})
	}

	SetVerbose(false)
	err := waitForNetworkState(appConfig{Nodes: nodes}, network.CompleteNetworkState)
	if err != nil {
		return cancelFunc, throw.W(err, "Can't wait for NetworkState "+network.CompleteNetworkState.String())
	}
	return cancelFunc, nil
}
