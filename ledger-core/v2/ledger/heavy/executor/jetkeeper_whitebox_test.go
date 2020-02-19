// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package executor

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/store"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/stretchr/testify/require"
)

func BadgerDefaultOptions(dir string) badger.Options {
	ops := badger.DefaultOptions(dir)
	ops.CompactL0OnClose = false
	ops.SyncWrites = false

	return ops
}

func initDB(t *testing.T, testPulse insolar.PulseNumber) (JetKeeper, string, *store.BadgerDB, *jet.DBStore, *pulse.DB) {
	ctx := inslogger.TestContext(t)
	tmpdir, err := ioutil.TempDir("", "bdb-test-")

	require.NoError(t, err)

	ops := BadgerDefaultOptions(tmpdir)
	db, err := store.NewBadgerDB(ops)
	require.NoError(t, err)

	jets := jet.NewDBStore(db)
	pulses := pulse.NewDB(db)
	err = pulses.Append(ctx, insolar.Pulse{PulseNumber: insolar.GenesisPulse.PulseNumber})
	require.NoError(t, err)

	err = pulses.Append(ctx, insolar.Pulse{PulseNumber: testPulse})
	require.NoError(t, err)

	jetKeeper := NewJetKeeper(jets, db, pulses)

	return jetKeeper, tmpdir, db, jets, pulses
}

func Test_JetKeeperKey(t *testing.T) {
	k := jetKeeperKey(insolar.GenesisPulse.PulseNumber)
	d := k.ID()
	require.Equal(t, k, newJetKeeperKey(d))
}

func Test_TruncateHead(t *testing.T) {
	ctx := inslogger.TestContext(t)
	testPulse := insolar.GenesisPulse.PulseNumber + 10
	ji, tmpDir, db, jets, _ := initDB(t, testPulse)
	defer os.RemoveAll(tmpDir)
	defer db.Stop(ctx)

	testJet := insolar.ZeroJetID

	err := jets.Update(ctx, testPulse, true, testJet)
	require.NoError(t, err)
	err = ji.AddHotConfirmation(ctx, testPulse, testJet, false)
	require.NoError(t, err)
	err = ji.AddDropConfirmation(ctx, testPulse, testJet, false)
	require.NoError(t, err)
	err = ji.AddBackupConfirmation(ctx, testPulse)
	require.NoError(t, err)

	require.Equal(t, testPulse, ji.TopSyncPulse())

	_, err = db.Get(jetKeeperKey(testPulse))
	require.NoError(t, err)

	nextPulse := testPulse + 10

	err = ji.AddDropConfirmation(ctx, nextPulse, gen.JetID(), false)
	require.NoError(t, err)
	err = ji.AddHotConfirmation(ctx, nextPulse, gen.JetID(), false)
	require.NoError(t, err)

	_, err = db.Get(jetKeeperKey(nextPulse))
	require.NoError(t, err)

	err = ji.(*DBJetKeeper).TruncateHead(ctx, nextPulse)
	require.NoError(t, err)

	_, err = db.Get(jetKeeperKey(testPulse))
	require.NoError(t, err)
	_, err = db.Get(jetKeeperKey(nextPulse))
	require.EqualError(t, err, "value not found")
}
