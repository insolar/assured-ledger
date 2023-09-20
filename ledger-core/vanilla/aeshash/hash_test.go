package aeshash

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHash(t *testing.T) {
	require.Equal(t, uint(0x9898989898989898), Str(""))
	require.Equal(t, uint(0xa0be041b1a8572be), Str("insolar"))
	require.Equal(t, uint(0xa0be041b1a8572be), ByteStr("insolar"))
	require.Equal(t, uint(0xa0be041b1a8572be), StrWithSeed("insolar", 0))
	require.Equal(t, uint(0xafa17a6f4a4ed3ef), StrWithSeed("insolar", 1))

	require.Equal(t, uint(0x9898989898989898), Slice(nil))
	require.Equal(t, uint(0x9898989898989898), Slice([]byte{}))
	require.Equal(t, uint(0xa0be041b1a8572be), Slice([]byte("insolar")))
	require.Equal(t, uint(0xa0be041b1a8572be), SliceWithSeed([]byte("insolar"), 0))
	require.Equal(t, uint(0xafa17a6f4a4ed3ef), SliceWithSeed([]byte("insolar"), 1))
}
