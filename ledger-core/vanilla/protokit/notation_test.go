package protokit

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentType(t *testing.T) {
	buf := []byte{0, 1, 99}
	reader := bytes.NewReader(buf)
	counters := [ContentPolymorph + 1]int{}

	for i := 256; i > 0; i-- {
		n := buf[0]
		ct, id, err := PeekContentTypeAndPolymorphIDFromBytes(buf)
		require.NoError(t, err, "0x%X", n)

		if n <= maskWireType {
			counters[ContentUndefined]++
			require.Equal(t, ContentUndefined, ct)
		} else {
			counters[ct]++
			switch ct {
			case ContentUndefined:
				switch WireType(n & maskWireType) {
				case WireVarint, WireFixed64, WireBytes, WireStartGroup, WireFixed32:
					require.Fail(t, "valid wire type", "0x%X", n)
				}
			case ContentText:
				require.True(t, n >= '\t', "0x%X", n)
				require.True(t, n < illegalUtf8FirstByte || n >= legalUtf8, "0x%X", n)
			case ContentBinary:
				require.True(t, n&maskWireType == 4, "0x%X", n)
				require.True(t, n >= illegalUtf8FirstByte && n < legalUtf8, "0x%X", n)
			case ContentMessage:
				u, c := DecodeVarintFromBytes(buf)
				require.NotZero(t, c)
				wt := WireTag(u)
				require.True(t, wt.IsValid())
				switch wt.Type() {
				case WireVarint, WireFixed64, WireBytes, WireStartGroup, WireFixed32:
				default:
					require.Fail(t, "invalid wire type", "0x%X", n)
				}
			case ContentPolymorph:
				require.True(t, n >= illegalUtf8FirstByte && n < legalUtf8, "0x%X", n)
				require.Equal(t, uint64(99), id)
			default:
				t.FailNow()
			}
		}

		reader.Reset(buf)
		pct, err := PeekPossibleContentTypes(reader)
		require.Equal(t, len(buf), reader.Len()) // must be unread

		require.NoError(t, err, "0x%X", n)
		require.Equal(t, PossibleContentTypes(buf[0]), pct, "0x%X", n)

		ct2, id2, err := PeekContentTypeAndPolymorphID(reader)
		require.NoError(t, err, "0x%X", n)
		require.Equal(t, ct, ct2, "0x%X", n)
		require.Equal(t, id, id2, "0x%X", n)

		buf[0]++
	}

	n := 0
	for _, c := range counters {
		n += c
	}
	require.Equal(t, 256, n)

	require.Equal(t, 24, counters[ContentUndefined])
	require.Equal(t, 183, counters[ContentText])
	require.Equal(t, 8, counters[ContentBinary])
	require.Equal(t, 33, counters[ContentMessage])
	require.Equal(t, 8, counters[ContentPolymorph])
}
