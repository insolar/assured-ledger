// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pulse

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/store"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulsar"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulsar/entropygenerator"
)

func BadgerDefaultOptions(dir string) badger.Options {
	ops := badger.DefaultOptions(dir)
	ops.CompactL0OnClose = false
	ops.SyncWrites = false

	return ops
}

func TestPulseKey(t *testing.T) {
	t.Parallel()

	expectedKey := pulseKey(insolar.GenesisPulse.PulseNumber)

	rawID := expectedKey.ID()

	actualKey := newPulseKey(rawID)
	require.Equal(t, expectedKey, actualKey)
}

func TestDropStorageDB_TruncateHead_NoSuchPulse(t *testing.T) {
	t.Parallel()

	ctx := inslogger.TestContext(t)
	tmpdir, err := ioutil.TempDir("", "bdb-test-")
	defer os.RemoveAll(tmpdir)
	assert.NoError(t, err)

	ops := BadgerDefaultOptions(tmpdir)
	dbMock, err := store.NewBadgerDB(ops)
	defer dbMock.Stop(ctx)
	require.NoError(t, err)

	pulseStore := NewDB(dbMock)

	err = pulseStore.TruncateHead(ctx, 77)
	require.NoError(t, err)
}

func TestDBStore_TruncateHead(t *testing.T) {
	t.Parallel()

	ctx := inslogger.TestContext(t)
	tmpdir, err := ioutil.TempDir("", "bdb-test-")
	defer os.RemoveAll(tmpdir)
	assert.NoError(t, err)

	ops := BadgerDefaultOptions(tmpdir)
	dbMock, err := store.NewBadgerDB(ops)
	defer dbMock.Stop(ctx)
	require.NoError(t, err)

	dbStore := NewDB(dbMock)

	numElements := 10

	startPulseNumber := insolar.GenesisPulse.PulseNumber
	for i := 0; i < numElements; i++ {
		pn := startPulseNumber + insolar.PulseNumber(i)
		pulse := *pulsar.NewPulse(0, pn, &entropygenerator.StandardEntropyGenerator{})
		err := dbStore.Append(ctx, pulse)
		require.NoError(t, err)
	}

	for i := 0; i < numElements; i++ {
		_, err := dbStore.ForPulseNumber(ctx, startPulseNumber+insolar.PulseNumber(i))
		require.NoError(t, err)
	}

	numLeftElements := numElements / 2
	err = dbStore.TruncateHead(ctx, startPulseNumber+insolar.PulseNumber(numLeftElements))
	require.NoError(t, err)

	for i := 0; i < numLeftElements; i++ {
		_, err := dbStore.ForPulseNumber(ctx, startPulseNumber+insolar.PulseNumber(i))
		require.NoError(t, err)
	}

	for i := numElements - 1; i >= numLeftElements; i-- {
		_, err := dbStore.ForPulseNumber(ctx, startPulseNumber+insolar.PulseNumber(i))
		require.EqualError(t, err, ErrNotFound.Error())
	}
}
