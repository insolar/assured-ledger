// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build functest,!cloud,cloud_with_consensus

package functest

import (
	"os"
	"strconv"
	"testing"

	"github.com/google/gops/agent"

	"github.com/insolar/assured-ledger/ledger-core/application/testutils/launchnet"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
)

func init() {
	// starts gops agent https://github.com/google/gops on default addr (127.0.0.1:0)
	if err := agent.Listen(agent.Options{}); err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	instestlogger.SetTestOutputWithStub()

	launchnet.SetCloudFileLogging(true)

	var (
		numVirtual = 5
		numLight   = 0
		numHeavy   = 0
		err        error
	)
	if numVirtualStr := os.Getenv("NUM_DISCOVERY_VIRTUAL_NODES"); numVirtualStr != "" {
		numVirtual, err = strconv.Atoi(numVirtualStr)
		if err != nil {
			panic(err)
		}
	}
	if numLightStr := os.Getenv("NUM_DISCOVERY_LIGHT_NODES"); numLightStr != "" {
		numLight, err = strconv.Atoi(numLightStr)
		if err != nil {
			panic(err)
		}
	}
	if numHeavyStr := os.Getenv("NUM_DISCOVERY_HEAVY_NODES"); numHeavyStr != "" {
		numHeavy, err = strconv.Atoi(numHeavyStr)
		if err != nil {
			panic(err)
		}
	}

	os.Exit(launchnet.RunCloudWithConsensus(numVirtual, numLight, numHeavy, func() int {
		return m.Run()
	}))
}
