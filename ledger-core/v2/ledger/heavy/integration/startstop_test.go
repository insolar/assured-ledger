// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build slowtest

package integration_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/v2/application"
)

func TestStartStop(t *testing.T) {
	cfg := DefaultHeavyConfig()
	defer os.RemoveAll(cfg.Ledger.Storage.DataDirectory)
	heavyConfig := application.GenesisHeavyConfig{
		ContractsConfig: application.GenesisContractsConfig{
			PKShardCount:          10,
			MAShardCount:          10,
			MigrationAddresses:    make([][]string, 10),
			VestingStepInPulses:   10,
			VestingPeriodInPulses: 10,
		},
	}
	s, err := NewServer(context.Background(), cfg, heavyConfig, nil)
	assert.NoError(t, err)
	s.Stop()
}
