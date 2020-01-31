package protokit

import (
	"bytes"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSafeWireTag(t *testing.T) {
	var test uint64
	test = math.MaxUint32 + 1
	wireTag, err := SafeWireTag(test)
	require.Error(t, err)
	require.Equal(t, WireTag(0), wireTag)

	test = uint64(math.Pow(2, lenWireType-1))
	wireTag, err = SafeWireTag(test)
	require.Error(t, err)
	require.Equal(t, WireTag(0), wireTag)

	test = uint64(math.Pow(2, lenWireType))
	wireTag, err = SafeWireTag(test)
	require.NoError(t, err)
	require.Equal(t, WireTag(test), wireTag)
}

func TestDecodeFromAnotherType(t *testing.T){
	buf := bytes.Buffer{}
	buf.Write([]byte{byte(math.Pow(2, lenWireType)), 123})

	// try to decode WreVariant to WireBytes
	_, _, err := WireBytes.DecodeFrom(&buf)
	require.Error(t, err)

	_, _, err = WireFixed64.DecodeFrom(&buf)
	require.Error(t, err)

	_, _, err = WireFixed32.DecodeFrom(&buf)
	require.Error(t, err)
}

func TestDecodeFrom(t *testing.T) {
	buf := bytes.Buffer{}

	// correct decode WireVarint
	type testCase struct {
		Name     string
		WireType WireType
		TypeByte byte
		Data     []byte
		Result   uint64
	}

	testCases := []testCase{
		{
			Name: "WireVarint",
			WireType: WireVarint,
			TypeByte: byte(math.Pow(2, lenWireType)),
			Data: []byte{123},
			Result: uint64(123),
		},
		{
			Name:     "WireFixed64",
			WireType: WireFixed64,
			TypeByte: byte(math.Pow(2, lenWireType) + 1),
			Data:     []byte{1, 1, 1, 1, 1, 1, 1, 1, 1},
			Result:   uint64(0x101010101010101),
		},
		{
			Name: "WireBytes",
			WireType: WireBytes,
			TypeByte: byte(math.Pow(2, lenWireType) + 2),
			Data: []byte{123},
			Result: uint64(123),
		},

		// TODO нет кейсов под эти данные, паника
		// {Name: "WireBytes", WireType: WireStartGroup, TypeByte: byte(math.Pow(2, lenWireType) + 3), Data: 123},
		// {Name: "WireBytes", WireType: WireEndGroup, TypeByte: byte(math.Pow(2, lenWireType) + 4), Data: 123},
		{
			Name:     "WireFixed32",
			WireType: WireFixed32,
			TypeByte: byte(math.Pow(2, lenWireType) + 5),
			Data:     []byte{1, 1, 1, 1, 1,},
			Result:   uint64(0x1010101),
		},

	}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {
			buf.Reset()
			buf.Write(append([]byte{test.TypeByte}, test.Data...))

			wireTag, data, err := test.WireType.DecodeFrom(&buf)
			require.NoError(t, err)
			require.Equal(t, test.Result, data)
			require.Equal(t, test.WireType, wireTag.Type())
		})
	}
}
