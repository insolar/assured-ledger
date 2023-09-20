package protokit

import (
	"bytes"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

type testCase struct {
	Name     string
	WireType WireType
	FieldId  float64
	TypeByte byte
	Data     []byte
	Result   uint64
	Size     uint64
}

func (tc *testCase) getTypeByte() byte {
	return byte(math.Pow(2, WireTypeBits) + tc.FieldId)
}

// TODO this cases cause panics - WireStartGroup, WireEndGroup
var testCases = []testCase{
	{
		Name:     "WireVarint",
		WireType: WireVarint,
		FieldId:  0,
		Data:     []byte{123},
		Result:   uint64(123),
		Size:     2,
	},
	{
		Name:     "WireFixed64",
		WireType: WireFixed64,
		FieldId:  1,
		Data:     []byte{1, 1, 1, 1, 1, 1, 1, 1, 1},
		Result:   uint64(0x101010101010101),
		Size:     9,
	},
	{
		Name:     "WireBytes",
		WireType: WireBytes,
		FieldId:  2,
		Data:     []byte{123},
		Result:   uint64(123),
		Size:     2,
	},
	{
		Name:     "WireFixed32",
		WireType: WireFixed32,
		FieldId:  5,
		Data:     []byte{1, 1, 1, 1, 1},
		Result:   uint64(0x1010101),
		Size:     5,
	},
}

func TestSafeWireTag(t *testing.T) {
	var test uint64
	test = math.MaxUint32 + 1
	wireTag, err := SafeWireTag(test)
	require.Error(t, err)
	require.Equal(t, WireTag(0), wireTag)

	test = uint64(math.Pow(2, WireTypeBits-1))
	wireTag, err = SafeWireTag(test)
	require.Error(t, err)
	require.Equal(t, WireTag(0), wireTag)

	test = uint64(math.Pow(2, WireTypeBits))
	wireTag, err = SafeWireTag(test)
	require.NoError(t, err)
	require.Equal(t, WireTag(test), wireTag)
	require.Equal(t, WireVarint, wireTag.Type())
}

func TestWireTypeDecodeFromAnotherType(t *testing.T) {
	buf := bytes.Buffer{}
	buf.Write([]byte{byte(math.Pow(2, WireTypeBits)), 123})

	// try to decode WireVariant to WireBytes
	_, _, err := WireBytes.ReadTagValue(&buf)
	require.Error(t, err)

	// try to decode WireVariant to WireFixed64
	_, _, err = WireFixed64.ReadTagValue(&buf)
	require.Error(t, err)

	// try to decode WireVariant to WireFixed32
	_, _, err = WireFixed32.ReadTagValue(&buf)
	require.Error(t, err)
}

func TestWireTypeDecodeFrom(t *testing.T) {
	buf := bytes.Buffer{}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {
			buf.Reset()
			buf.Write(append([]byte{test.getTypeByte()}, test.Data...))

			wireTag, data, err := test.WireType.ReadTagValue(&buf)
			require.NoError(t, err)
			require.Equal(t, test.Result, data)
			require.Equal(t, test.WireType, wireTag.Type())
		})
	}
}

func TestWireTagDecodeFromAnotherType(t *testing.T) {
	buf := bytes.Buffer{}
	buf.Write([]byte{byte(math.Pow(2, WireTypeBits)), 123})

	// try to decode WireVariant to WireBytes
	_, err := WireBytes.Tag(1).ReadTagValue(&buf)
	require.Error(t, err)

	// try to decode WireVariant to WireFixed64
	_, err = WireFixed64.Tag(1).ReadTagValue(&buf)
	require.Error(t, err)

	// try to decode WireVariant to WireFixed32
	_, err = WireFixed32.Tag(1).ReadTagValue(&buf)
	require.Error(t, err)
}

func TestWireTagDecodeFrom(t *testing.T) {
	buf := bytes.Buffer{}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {
			buf.Reset()
			buf.Write(append([]byte{test.getTypeByte()}, test.Data...))

			tag := test.WireType.Tag(1)
			data, err := tag.ReadTagValue(&buf)
			require.NoError(t, err)
			require.Equal(t, test.Result, data)
		})
	}
}

func TestEnsureFixedFieldSize(t *testing.T) {
	tag := WireFixed64.Tag(1)
	newTag := tag.EnsureFixedFieldSize(9)
	require.Equal(t, tag, newTag)

	tag = WireFixed32.Tag(1)
	newTag = tag.EnsureFixedFieldSize(5)
	require.Equal(t, tag, newTag)
}

func TestEnsureFixedFieldSizeFailed(t *testing.T) {
	tag := WireVarint.Tag(1)
	require.PanicsWithValue(t, "illegal state - not fixed size", func() { tag.EnsureFixedFieldSize(12) })

	tag = WireBytes.Tag(1)
	require.PanicsWithValue(t, "illegal state - not fixed size", func() { tag.EnsureFixedFieldSize(12) })
}

func TestWireTagMustEncodeTo(t *testing.T) {
	buf := bytes.Buffer{}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {
			buf.Reset()
			buf.Write(append([]byte{test.getTypeByte()}, test.Data...))

			tag := test.WireType.Tag(1)
			tag.MustWrite(&buf, 1234)
		})
	}
}

func TestWireTagFieldSize(t *testing.T) {
	buf := bytes.Buffer{}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {
			buf.Reset()
			buf.Write(append([]byte{test.getTypeByte()}, test.Data...))

			tag := test.WireType.Tag(1)
			size := tag.FieldSize(0)
			require.Equal(t, test.Size, size)
		})
	}
}

func TestSizeTag(t *testing.T) {
	require.Equal(t, 1, SizeTag(1))
	require.Panics(t, func() { SizeTag(0) })
	require.Panics(t, func() { SizeTag(MaxFieldID + 1) })
}

func TestEncodeBytes(t *testing.T) {
	for i := uint64(1); i != 0; i <<= 1 {
		var b [MaxVarintSize]byte
		n := EncodeVarintToBytes(b[:], i)
		require.Equal(t, n, SizeVarint64(i))
		x, n2 := DecodeVarintFromBytes(b[:])
		if n != n2 {
			DecodeVarintFromBytes(b[:])
		}
		require.Equal(t, n, n2)
		require.Equal(t, i, x)
	}
}

func TestSizeVarint(t *testing.T) {
	sz := 1
	for i := uint32(1); i != 0; i <<= 1 {
		n := SizeVarint32(i)
		require.GreaterOrEqual(t, n, sz)
		require.Equal(t, n, SizeVarint64(uint64(i)))
		if n == sz {
			continue
		}
		require.Equal(t, sz+1, n)
		require.Equal(t, sz, SizeVarint32(i-1), i)
		sz = n
	}

	for i := uint64(1 << 32); i != 0; i <<= 1 {
		n := SizeVarint64(i)
		require.GreaterOrEqual(t, n, sz)
		if n == sz {
			continue
		}
		require.Equal(t, sz+1, n)
		require.Equal(t, sz, SizeVarint64(i-1), i)
		sz = n
	}
	require.Equal(t, MaxVarintSize, sz)
}
