// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package launchnet

import (
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/log"
	"github.com/insolar/assured-ledger/ledger-core/log/global"
	"github.com/insolar/assured-ledger/ledger-core/server"
	"github.com/insolar/assured-ledger/ledger-core/testutils/cloud"
)

// this is not actually test, but it provides convenient way to start cloud network with breakpoints
func Test_RunCloud(t *testing.T) {
	t.Skip()
	var (
		numVirtual        = uint(10)
		numLightMaterials = uint(0)
		numHeavyMaterials = uint(0)
	)

	majorityRule := int(numVirtual + numLightMaterials + numHeavyMaterials)

	nodeConfiguration := cloud.NodeConfiguration{
		Virtual:       numVirtual,
		HeavyMaterial: numHeavyMaterials,
		LightMaterial: numLightMaterials,
	}

	signChan := make(chan os.Signal, 1)
	signal.Notify(signChan, os.Interrupt, syscall.SIGTERM)

	cloudSettings := cloud.Settings{
		Running:      nodeConfiguration,
		Prepared:     nodeConfiguration,
		MinRoles:     nodeConfiguration,
		MajorityRule: majorityRule,

		Log:    struct{ Level string }{Level: log.DebugLevel.String()},
		Pulsar: struct{ PulseTime int }{PulseTime: GetPulseTime()},
	}

	confProvider := cloud.NewConfigurationProvider(cloudSettings)

	s := server.NewMultiServer(confProvider)
	go func() {
		sig := <-signChan

		global.Infof("%v signal received\n", sig)

		s.Stop()
	}()

	s.Serve()
}
