package reference

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBinarySizeGlobal(t *testing.T) {
	testWalkLocal(t, func(t *testing.T, base Local) {
		testWalkLocal(t, func(t *testing.T, local Local) {
			g := New(base, local)
			gString := g.String()
			if base.IsEmpty() {
				if local.IsEmpty() {
					require.Zero(t, BinarySize(g), gString)
				} else {
					require.Equal(t, LocalBinarySize+LocalBinaryPulseAndScopeSize, BinarySize(g), gString)
				}
				return
			}

			if base == local {
				require.Equal(t, BinarySizeLocal(base), BinarySize(g), gString)
			} else {
				require.Equal(t, LocalBinarySize+BinarySizeLocal(base), BinarySize(g), gString)
			}
		})
	})
}

func TestBinaryMarshal(t *testing.T) {
	testWalkLocal(t, func(t *testing.T, base Local) {
		testWalkLocal(t, func(t *testing.T, local Local) {
			g := New(base, local)

			gString := g.String()

			expected := BinarySize(g)

			b, err := Marshal(g)
			require.NoError(t, err, gString)
			require.Equal(t, expected, len(b), gString)

			testUnmarshalGlobal(t, g, b)

			b = make([]byte, GlobalBinarySize+4)

			n, err := MarshalTo(g, b)
			require.NoError(t, err, gString)
			require.Equal(t, expected, n, gString)

			testUnmarshalGlobal(t, g, b[:n])

			for i := range b {
				b[i] = 0
			}

			n, err = MarshalToSizedBuffer(g, b)
			require.NoError(t, err, gString)
			require.Equal(t, expected, n, gString)

			testUnmarshalGlobal(t, g, b[len(b)-n:])
		})
	})
}

func testUnmarshalGlobal(t *testing.T, g Global, b []byte) {
	r, err := UnmarshalGlobal(b)
	require.NoError(t, err)
	require.Equal(t, g, r)

	b = append(b, 0)
	r, err = UnmarshalGlobal(b)
	require.Error(t, err)
	require.Equal(t, Global{}, r)
}

func TestBinaryMarshalJSON(t *testing.T) {
	testWalkLocal(t, func(t *testing.T, base Local) {
		testWalkLocal(t, func(t *testing.T, local Local) {
			g := New(base, local)
			gString := g.String()

			b, err := MarshalJSON(g)
			require.NoError(t, err, gString)

			var r Global
			err = _unmarshalJSON(&r, b, NewDefaultDecoder(AllowRecords|DoNotVerify))
			require.NoError(t, err, gString)

			require.Equal(t, g, r, string(b))
		})
	})
}
