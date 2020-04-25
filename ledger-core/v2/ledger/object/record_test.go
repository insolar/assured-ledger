// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package object

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
)

func TestRecordKey(t *testing.T) {
	t.Parallel()

	expectedKey := recordKey(gen.ID())

	rawID := expectedKey.ID()

	actualKey := newRecordKey(rawID)
	require.Equal(t, expectedKey, actualKey)
}

func TestRecordPositionKey(t *testing.T) {
	t.Parallel()

	expectedKey := recordPositionKey{pn: insolar.GenesisPulse.PulseNumber, number: 42}

	rawID := expectedKey.ID()

	actualKey := newRecordPositionKey(insolar.GenesisPulse.PulseNumber, 42)

	actualKeyFromBytes := newRecordPositionKeyFromBytes(rawID)
	require.Equal(t, expectedKey, actualKeyFromBytes)
	require.Equal(t, expectedKey, actualKey)
}

func TestLastKnownRecordPositionKey(t *testing.T) {
	t.Parallel()

	expectedKey := lastKnownRecordPositionKey{pn: insolar.GenesisPulse.PulseNumber}

	rawID := expectedKey.ID()

	actualKey := newLastKnownRecordPositionKey(rawID)
	require.Equal(t, expectedKey, actualKey)
}
