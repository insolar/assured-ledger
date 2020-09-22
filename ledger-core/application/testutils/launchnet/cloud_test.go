// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package launchnet

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/server"
)

// these are not actually tests, but it provides convenient way to start cloud network with breakpoints
func Test_RunCloud(t *testing.T) {
	t.Skip()
	var (
		numVirtual        = 10
		numLightMaterials = 0
		numHeavyMaterials = 0
	)

	cloudSettings := CloudSettings{Virtual: numVirtual, Light: numLightMaterials, Heavy: numHeavyMaterials}

	appConfigs, cloudBaseConf, certFactory, keyFactory := PrepareCloudConfiguration(cloudSettings)
	baseConfig := configuration.Configuration{}
	baseConfig.Log = cloudBaseConf.Log

	confProvider := &server.CloudConfigurationProvider{
		PulsarConfig:       cloudBaseConf.PulsarConfiguration,
		BaseConfig:         baseConfig,
		CertificateFactory: certFactory,
		KeyFactory:         keyFactory,
		GetAppConfigs: func() []configuration.Configuration {
			return appConfigs
		},
	}

	s := server.NewMultiServer(confProvider)
	s.Serve()
}

func Test_RunCloud_With_ManualPulsar(t *testing.T) {
	t.Skip()
	var (
		numVirtual        = 10
		numLightMaterials = 0
		numHeavyMaterials = 0
	)

	cloudSettings := CloudSettings{Virtual: numVirtual, Light: numLightMaterials, Heavy: numHeavyMaterials}

	appConfigs, cloudBaseConf, certFactory, keyFactory := PrepareCloudConfiguration(cloudSettings)
	baseConfig := configuration.Configuration{}
	baseConfig.Log = cloudBaseConf.Log

	confProvider := &server.CloudConfigurationProvider{
		PulsarConfig:       cloudBaseConf.PulsarConfiguration,
		BaseConfig:         baseConfig,
		CertificateFactory: certFactory,
		KeyFactory:         keyFactory,
		GetAppConfigs: func() []configuration.Configuration {
			return appConfigs
		},
	}

	s, pulsar := server.NewMultiServerWithCustomPulsar(confProvider)
	go func() {
		s.Serve()
	}()

	// wait for starting all components
	for !s.(*insapp.Server).Started() {
		time.Sleep(time.Millisecond)
	}

	pulsar.IncrementPulse()
	pulsar.IncrementPulse()

	for i := 0; i < 3; i++ {
		for i := 0; i < len(appConfigs); i++ {
			cert, err := certFactory(nil, nil, appConfigs[i].CertificatePath)
			if err != nil {
				panic(err)
			}
			pulsar.IncrementPulseForNode(context.Background(), cert.GetCertificate().GetNodeRef())
		}
		fmt.Println("Pulses on all nodes are changed")
	}
}
