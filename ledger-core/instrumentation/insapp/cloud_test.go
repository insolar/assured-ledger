// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insapp_test

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/configuration"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/insapp"
	"github.com/insolar/assured-ledger/ledger-core/server"
)

func Test_RunCloud(t *testing.T) {
	t.Skip()
	var (
		numVirtual        = 10
		numLightMaterials = 0
		numHeavyMaterials = 0
	)

	appConfigs, baseConf, certFactory, keyFactory := insapp.PrepareCloudConfiguration(numVirtual, numLightMaterials, numHeavyMaterials)

	confProvider := insapp.CloudConfigurationProvider{
		CloudConfig:        baseConf,
		CertificateFactory: certFactory,
		KeyFactory:         keyFactory,
		GetAppConfigs: func() []configuration.Configuration {
			return appConfigs
		},
	}

	s := server.NewMultiServer(confProvider)
	s.Serve()
}
