package lineage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecordTypeSet(t *testing.T) {
	rts := RecordTypeSet{}
	require.True(t, rts.IsZero())

	rts = setOf(1, 999)
	require.True(t, rts.Has(1))
	require.True(t, rts.Has(999))

	for i := 1000; i < 1100; i++ {
		require.False(t, rts.Has(RecordType(i)))
	}

	for i := 2; i < 999; i++ {
		require.False(t, rts.Has(RecordType(i)))
	}
	require.False(t, rts.Has(0))

	require.Equal(t, 125, len(rts.mask))
	require.Equal(t, 0, int(rts.ofs))

	rts = setOf(900, 999)
	require.True(t, rts.Has(900))
	require.True(t, rts.Has(999))

	for i := 1000; i < 1100; i++ {
		require.False(t, rts.Has(RecordType(i)))
	}

	for i := 0; i < 900; i++ {
		require.False(t, rts.Has(RecordType(i)))
	}

	for i := 901; i < 999; i++ {
		require.False(t, rts.Has(RecordType(i)))
	}

	require.Equal(t, 13, len(rts.mask))
	require.Equal(t, 896, int(rts.ofs))
}
