// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

// +build slowtest

package mimic

import (
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/application"
	"github.com/insolar/assured-ledger/ledger-core/v2/application/genesis"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/drop"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/object"
	"github.com/insolar/assured-ledger/ledger-core/v2/platformpolicy"
)

func TestMimicLedger_Genesis(t *testing.T) {
	cleanup, bootstrapDir, err := GenerateBootstrap(true)
	require.NoError(t, err)
	defer cleanup()

	senderMock := bus.NewSenderMock(t)

	t.Run("WithMocks", func(t *testing.T) {
		// t.Parallel()
		ctx := inslogger.TestContext(t)

		mc := minimock.NewController(t)

		pcs := platformpolicy.NewPlatformCryptographyScheme()
		pulseStorage := pulse.NewStorageMem()
		dmm := drop.NewModifierMock(mc).SetMock.Return(nil)
		imm := object.NewIndexModifierMock(mc).
			SetIndexMock.Return(nil).
			UpdateLastKnownPulseMock.Return(nil)
		rmm := object.NewRecordModifierMock(mc).SetMock.Return(nil)

		mimicLedgerInstance := NewMimicLedger(ctx, pcs, pulseStorage, pulseStorage, senderMock)
		mimicStorage := mimicLedgerInstance.(*mimicLedger).storage

		mimicClient := NewClient(mimicStorage)

		genesisContractsConfig, err := ReadGenesisContractsConfig(bootstrapDir)
		require.NoError(t, err)

		genesisObject := genesis.Genesis{
			ArtifactManager: mimicClient,
			BaseRecord: &genesis.BaseRecord{
				DB:             mimicStorage,
				DropModifier:   dmm,
				PulseAppender:  pulseStorage,
				PulseAccessor:  pulseStorage,
				RecordModifier: rmm,
				IndexModifier:  imm,
			},
			DiscoveryNodes:  []application.DiscoveryNodeRegister{},
			ContractsConfig: *genesisContractsConfig,
		}

		err = genesisObject.Start(ctx)
		assert.NoError(t, err)
	})

	t.Run("WithoutMocks", func(t *testing.T) {
		// t.Parallel()
		ctx := inslogger.TestContext(t)

		pcs := platformpolicy.NewPlatformCryptographyScheme()
		pulseStorage := pulse.NewStorageMem()

		mimicLedgerInstance := NewMimicLedger(ctx, pcs, pulseStorage, pulseStorage, senderMock)

		err := mimicLedgerInstance.LoadGenesis(ctx, bootstrapDir)
		assert.NoError(t, err)
	})
}
