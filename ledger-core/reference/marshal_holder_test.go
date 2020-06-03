// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBinarySizeGlobal(t *testing.T) {
	testWalkLocal(t, func(t *testing.T, base Local) {
		testWalkLocal(t, func(t *testing.T, local Local) {
			g := New(base, local)
			if base.IsEmpty() {
				if local.IsEmpty() {
					require.Zero(t, BinarySize(g), g.String())
				} else {
					require.Equal(t, LocalBinarySize+LocalBinaryPulseAndScopeSize, BinarySize(g), g.String())
				}
				return
			}

			if base == local {
				require.Equal(t, BinarySizeLocal(base), BinarySize(g), g.String())
			} else {
				require.Equal(t, LocalBinarySize+BinarySizeLocal(base), BinarySize(g), g.String())
			}
		})
	})
}

func TestBinaryMarshal(t *testing.T) {
	testWalkLocal(t, func(t *testing.T, base Local) {
		testWalkLocal(t, func(t *testing.T, local Local) {
			g := New(base, local)

			expected := BinarySize(g)

			b, err := Marshal(g)
			require.NoError(t, err, g.String())
			require.Equal(t, expected, len(b), g.String())

			testUnmarshalGlobal(t, g, b)

			b = make([]byte, GlobalBinarySize+4)

			n, err := MarshalTo(g, b)
			require.NoError(t, err, g.String())
			require.Equal(t, expected, n, g.String())

			testUnmarshalGlobal(t, g, b[:n])

			for i := range b {
				b[i] = 0
			}

			n, err = MarshalToSizedBuffer(g, b)
			require.NoError(t, err, g.String())
			require.Equal(t, expected, n, g.String())

			testUnmarshalGlobal(t, g, b[len(b)-n:])
		})
	})
}

func testUnmarshalGlobal(t *testing.T, g Global, b []byte) {
	r, err := UnmarshalGlobal(b)
	require.NoError(t, err, g.String())
	require.Equal(t, g, r, g.String())

	b = append(b, 0)
	r, err = UnmarshalGlobal(b)
	require.Error(t, err, g.String())
	require.Equal(t, Global{}, r, g.String())
}

func TestBinaryMarshalJSON(t *testing.T) {
	testWalkLocal(t, func(t *testing.T, base Local) {
		testWalkLocal(t, func(t *testing.T, local Local) {
			g := New(base, local)

			b, err := MarshalJSON(g)
			require.NoError(t, err, g.String())

			var r Global
			err = _unmarshalJSON(&r, b, NewDefaultDecoder(AllowRecords|DoNotVerify))
			require.NoError(t, err, g.String())

			require.Equal(t, g, r, string(b))
		})
	})
}
