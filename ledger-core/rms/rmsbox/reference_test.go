package rmsbox

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/reference"
)

func TestReferenceZeroOrEmpty(t *testing.T) {
	b := make([]byte, 64)

	ref := Reference{}
	require.True(t, ref.IsZero())
	require.True(t, ref.IsEmpty())
	n, err := ref.MarshalTo(b)
	require.NoError(t, err)
	require.Zero(t, n)

	ref.Set(reference.Global{})
	require.True(t, ref.IsZero())
	require.True(t, ref.IsEmpty())
	n, err = ref.MarshalTo(b)
	require.NoError(t, err)
	require.Zero(t, n)

	ref.Set(nil)
	require.True(t, ref.IsZero())
	require.True(t, ref.IsEmpty())
	n, err = ref.MarshalTo(b)
	require.NoError(t, err)
	require.Zero(t, n)

	ref.SetExact(reference.Global{})
	require.False(t, ref.IsZero())
	require.True(t, ref.IsEmpty())
	n, err = ref.MarshalTo(b)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	ref2 := Reference{}
	err = ref2.Unmarshal(b[:n])
	require.NoError(t, err)
	require.False(t, ref.IsZero())
	require.True(t, ref.IsEmpty())

	ref.SetExact(nil)
	require.True(t, ref.IsZero())
	require.True(t, ref.IsEmpty())
	n, err = ref.MarshalTo(b)
	require.NoError(t, err)
	require.Zero(t, n)

	ref2 = Reference{}
	err = ref2.Unmarshal(b[:0])
	require.NoError(t, err)
	require.True(t, ref.IsZero())
	require.True(t, ref.IsEmpty())
}
