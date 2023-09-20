package reference

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
)

func TestBinarySizeLocal(t *testing.T) {
	require.Equal(t, 0, BinarySizeLocal(Local{}))

	for _, i := range []pulse.Number{1, 255, 256, 267, pulse.LocalRelative - 1} {
		require.Equal(t, LocalBinaryPulseAndScopeSize, BinarySizeLocal(NewLocal(i, 0, LocalHash{})))
		require.Equal(t, LocalBinaryPulseAndScopeSize, BinarySizeLocal(NewLocal(i, SubScopeGlobal, LocalHash{})))

		for j := LocalBinaryHashSize; j > 0; j-- {
			local := Local{pulseAndScope: NewLocalHeader(i, 0)}
			local.hash[j-1] = 1
			require.Equal(t, LocalBinaryPulseAndScopeSize+j, BinarySizeLocal(local))
		}
	}

	require.Equal(t, LocalBinarySize, BinarySizeLocal(NewLocal(pulse.LocalRelative, 0, LocalHash{})))
	require.Equal(t, LocalBinarySize, BinarySizeLocal(NewLocal(pulse.LocalRelative, 0, hashLocal())))

	require.Equal(t, LocalBinarySize, BinarySizeLocal(NewLocal(pulse.MinTimePulse, 0, LocalHash{})))
	require.Equal(t, LocalBinarySize, BinarySizeLocal(NewLocal(pulse.MinTimePulse, 0, hashLocal())))

	require.Equal(t, LocalBinarySize, BinarySizeLocal(NewLocal(pulse.MaxTimePulse, 0, LocalHash{})))
	require.Equal(t, LocalBinarySize, BinarySizeLocal(NewLocal(pulse.MaxTimePulse, 0, hashLocal())))

	require.Equal(t, LocalBinarySize, BinarySizeLocal(Local{pulseAndScope: math.MaxUint32}))
}

func testWalkLocal(t *testing.T, testFn func(*testing.T, Local)) {
	testFn(t, Local{})

	for _, i := range []pulse.Number{1, 255, 256, 267, pulse.LocalRelative - 1} {
		testFn(t, NewLocal(i, 0, LocalHash{}))
		testFn(t, NewLocal(i, SubScopeGlobal, LocalHash{}))

		for j := LocalBinaryHashSize; j > 0; j-- {
			local := Local{pulseAndScope: NewLocalHeader(i, 0)}
			local.hash[j-1] = 1
			testFn(t, local)
		}
	}

	testFn(t, NewLocal(pulse.LocalRelative, 0, LocalHash{}))
	testFn(t, NewLocal(pulse.LocalRelative, 0, hashLocal()))

	testFn(t, NewLocal(pulse.MinTimePulse, 0, LocalHash{}))
	testFn(t, NewLocal(pulse.MinTimePulse, 0, hashLocal()))

	testFn(t, NewLocal(pulse.MaxTimePulse, 0, LocalHash{}))
	testFn(t, NewLocal(pulse.MaxTimePulse, 0, hashLocal()))

	testFn(t, Local{pulseAndScope: math.MaxUint32})
}

func TestMarshalLocal(t *testing.T) {
	testWalkLocal(t, testMarshalLocal)
}

func testMarshalLocal(t *testing.T, local Local) {
	expected := BinarySizeLocal(local)

	b, err := MarshalLocal(local)
	require.NoError(t, err, local.String())
	require.Equal(t, expected, len(b), local.String())

	testUnmarshalLocal(t, local, b)

	b = make([]byte, LocalBinarySize+4)

	n, err := MarshalLocalTo(local, b)
	require.NoError(t, err, local.String())
	require.Equal(t, expected, n, local.String())

	testUnmarshalLocal(t, local, b[:n])

	for i := range b {
		b[i] = 0
	}

	n, err = MarshalLocalToSizedBuffer(local, b)
	require.NoError(t, err, local.String())
	require.Equal(t, expected, n, local.String())

	testUnmarshalLocal(t, local, b[len(b)-n:])

	for i := range b {
		b[i] = 0
	}

	WriteWholeLocalTo(local, b)
	require.Equal(t, local, ReadWholeLocalFrom(b))
}

func testUnmarshalLocal(t *testing.T, local Local, b []byte) {
	r, err := UnmarshalLocal(b)
	require.NoError(t, err, local.String())
	require.Equal(t, local, r, local.String())

	b = append(b, 0)
	r, err = UnmarshalLocal(b)
	require.Error(t, err, local.String())
	require.Equal(t, Local{}, r, local.String())

	for i := len(b) - 1; i > 1; i-- {
		r, err = UnmarshalLocal(b[:i-1])
		if i == LocalBinaryPulseAndScopeSize+1 && local.Pulse() < pulse.LocalRelative {
			require.NoError(t, err, local.String())
			require.Equal(t, local.WithHash(LocalHash{}), r, local.String())
			continue
		}
		require.Error(t, err, local.String())
		require.Equal(t, Local{}, r, local.String())
	}
}

func TestMarshalLocalJSON(t *testing.T) {
	testWalkLocal(t, func(t *testing.T, local Local) {
		b, err := MarshalLocalJSON(local)
		require.NoError(t, err, local.String())
		r, err := UnmarshalLocalJSON(b)
		require.NoError(t, err, local.String())
		require.Equal(t, local, r, local.String())
	})
}
