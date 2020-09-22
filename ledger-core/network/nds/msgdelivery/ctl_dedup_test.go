// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeduplicator_Linear(t *testing.T) {
	dd := receiveDeduplicator{}
	id := DedupID(0)

	require.False(t, dd.Add(0))
	require.False(t, dd.Has(0))

	for i := minReceiveWindow; i > 0; i-- {
		id++
		require.False(t, dd.Has(id))
		require.True(t, dd.Add(id))
		require.True(t, dd.Has(id))
		require.False(t, dd.Add(id))
	}

	require.Equal(t, id, dd.maxReceived)
	require.Equal(t, id, dd.minReceived)
	require.Equal(t, id, dd.maxContinuous)
}

func TestDeduplicator_SmallGap(t *testing.T) {
	dd := receiveDeduplicator{}
	id := DedupID(0)

	for i := minReceiveWindow; i > 0; i-- {
		id++
		require.True(t, dd.Add(id))
	}

	beforeGap := id
	id += 2
	afterGap := id + 1

	for i := maxReceiveWindow - 2; i > 0; i-- {
		id++
		require.True(t, dd.Add(id))
		require.Equal(t, id, dd.maxReceived)
		require.Equal(t, beforeGap, dd.minReceived)
		require.Equal(t, beforeGap, dd.maxContinuous)
	}

	for i := maxReceiveWindow; i > 0; i-- {
		id++
		require.True(t, dd.Add(id))
		require.Equal(t, id, dd.maxReceived)
		require.Equal(t, id, dd.minReceived)
		require.Equal(t, beforeGap, dd.maxContinuous)
	}

	for i := id; i > 0; i-- {
		if i >= beforeGap+1 && i < afterGap {
			require.False(t, dd.Has(i), i)
		} else {
			require.True(t, dd.Has(i), i)
		}
	}

	require.True(t, dd.Add(afterGap-1))
	require.Equal(t, id, dd.maxReceived)
	require.Equal(t, id, dd.minReceived)
	require.Equal(t, beforeGap, dd.maxContinuous)

	require.True(t, dd.Add(afterGap-2))
	require.Equal(t, id, dd.maxReceived)
	require.Equal(t, id, dd.minReceived)
	require.Equal(t, id, dd.maxContinuous)

	for i := id; i > 0; i-- {
		require.True(t, dd.Has(i), i)
	}
}

func TestDeduplicator_Gaps(t *testing.T) {
	dd := receiveDeduplicator{}
	id := DedupID(0)

	for i := maxReceiveWindow << 4; i > 0; i-- {
		id += 3
		dd.Add(id)
		require.GreaterOrEqual(t, maxReceiveWindow, len(dd.received))
		require.GreaterOrEqual(t, maxReceiveExceptions, len(dd.excepts))
	}

	require.Equal(t, id, dd.maxReceived)
	require.GreaterOrEqual(t, maxReceiveWindow, len(dd.received))
	require.GreaterOrEqual(t, maxReceiveExceptions, len(dd.excepts))

	for i := maxReceiveWindow << 1; i > 2; i -= 3 {
		require.True(t, dd.Has(id), i)
		id--
		require.False(t, dd.Has(id), i)
		id--
		require.False(t, dd.Has(id), i)
		id--
	}

	id -= DedupID(maxReceiveWindow) // coverage in this area may vary

	for ; id > 0; id-- {
		if !dd.Has(id) {
			require.True(t, dd.Has(id), id)
		}
	}
}

func BenchmarkDeduplicator(b *testing.B) {
	b.Run("Linear", func(b *testing.B) {
		dd := receiveDeduplicator{}
		id := DedupID(0)

		for i := b.N; i > 0; i-- {
			id++
			dd.Add(id)
		}
	})

	b.Run("Gaps", func(b *testing.B) {
		dd := receiveDeduplicator{}
		id := DedupID(0)

		for i := b.N; i > 0; i-- {
			id += 2
			dd.Add(id)
		}
	})
}
