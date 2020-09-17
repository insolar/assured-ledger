// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package launchnet

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/server"
)

// this is not actually test, but it provides convenient way to start cloud network with breakpoints
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
