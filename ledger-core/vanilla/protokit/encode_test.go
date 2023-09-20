package protokit

import (
	"bytes"
	"io"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeVarint(t *testing.T) {
	lastSize := 1
	require.Equal(t, lastSize, testEncodeDecode(t, 0))
	require.Equal(t, lastSize, testEncodeDecode(t, 1))

	v := uint64(0x80)
	for v != 0 {
		n := testEncodeDecode(t, v-1)
		require.Equal(t, lastSize, n)

		lastSize++

		n = testEncodeDecode(t, v)
		require.Equal(t, lastSize, n)

		n = testEncodeDecode(t, v+1)
		require.Equal(t, lastSize, n)
		v <<= 7
	}
}

func testEncodeDecode(t *testing.T, v uint64) int {
	var b [MaxVarintSize]byte

	n := EncodeVarintToBytes(b[:], v)
	require.Equal(t, n, SizeVarint64(v))
	if v <= math.MaxUint32 {
		require.Equal(t, n, SizeVarint32(uint32(v)))
	}

	u, n2 := DecodeVarintFromBytes(b[:])
	require.Equal(t, n, n2)
	require.Equal(t, v, u)

	var err error
	u, n2, err = DecodeVarintFromBytesWithError(b[:])
	require.NoError(t, err)
	require.Equal(t, n, n2)
	require.Equal(t, v, u)

	u, n2, err = WireVarint.Tag(1).ReadValueFromBytes(b[:])
	require.NoError(t, err)
	require.Equal(t, n, n2)
	require.Equal(t, v, u)

	u, n2, err = WireBytes.Tag(1).ReadValueFromBytes(b[:])
	require.NoError(t, err)
	require.Equal(t, n, n2)
	require.Equal(t, v, u)

	for i := n - 1; i > 0; i-- {
		u, n2, err = DecodeVarintFromBytesWithError(b[:i])
		require.Equal(t, io.ErrUnexpectedEOF, err)
		require.Zero(t, n2)
		require.Zero(t, u)

		u, n2, err = WireVarint.Tag(1).ReadValueFromBytes(b[:i])
		require.Equal(t, io.ErrUnexpectedEOF, err)
		require.Zero(t, n2)
		require.Zero(t, u)

		u, n2, err = WireBytes.Tag(1).ReadValueFromBytes(b[:i])
		require.Equal(t, io.ErrUnexpectedEOF, err)
		require.Zero(t, n2)
		require.Zero(t, u)

		require.Panics(t, func() { DecodeVarintFromBytes(b[:i]) })
	}

	var b2 [MaxVarintSize]byte
	n2, err = EncodeVarintToBytesWithError(b2[:], v)
	require.NoError(t, err)
	require.Equal(t, b[:n], b2[:n2])

	n2, err = WireVarint.Tag(1).WriteValueToBytes(b[:], v)
	require.NoError(t, err)
	require.Equal(t, b[:n], b2[:n2])

	n2, err = WireBytes.Tag(1).WriteValueToBytes(b[:], v)
	require.NoError(t, err)
	require.Equal(t, b[:n], b2[:n2])

	for i := n - 1; i > 0; i-- {
		n2, err = EncodeVarintToBytesWithError(b2[:i], v)
		require.Equal(t, io.ErrShortBuffer, err)
		require.Zero(t, n2)

		n2, err = WireVarint.Tag(1).WriteValueToBytes(b[:i], v)
		require.Equal(t, io.ErrShortBuffer, err)
		require.Zero(t, n2)

		n2, err = WireBytes.Tag(1).WriteValueToBytes(b[:i], v)
		require.Equal(t, io.ErrShortBuffer, err)
		require.Zero(t, n2)
	}

	buf := &bytes.Buffer{}

	err = EncodeVarint(buf, v)
	require.NoError(t, err)
	bufCopy := buf.Bytes()

	u, err = DecodeVarint(buf)
	require.NoError(t, err)
	require.Equal(t, v, u)
	require.Zero(t, buf.Len())

	buf = &bytes.Buffer{}
	err = WireVarint.Tag(1).WriteValue(buf, v)
	require.NoError(t, err)
	require.Equal(t, bufCopy, buf.Bytes())

	u, err = WireVarint.Tag(1).ReadValue(buf)
	require.NoError(t, err)
	require.Equal(t, v, u)
	require.Zero(t, buf.Len())

	buf = &bytes.Buffer{}
	err = WireBytes.Tag(1).WriteValue(buf, v)
	require.NoError(t, err)
	require.Equal(t, bufCopy, buf.Bytes())

	u, err = WireBytes.Tag(1).ReadValue(buf)
	require.NoError(t, err)
	require.Equal(t, v, u)
	require.Zero(t, buf.Len())

	return n
}

func TestEncodeDecodeFixed64(t *testing.T) {
	v := uint64(1)
	for {
		buf := &bytes.Buffer{}
		err := EncodeFixed64(buf, v)
		require.NoError(t, err)

		u, n, err := DecodeFixed64FromBytesWithError(buf.Bytes())
		require.NoError(t, err)
		require.Equal(t, v, u)
		require.Equal(t, buf.Len(), n)

		b := make([]byte, n)
		n, err = EncodeFixed64ToBytes(b, v)
		require.NoError(t, err)
		require.Equal(t, buf.Bytes(), b[:n])

		n, err = EncodeFixed64ToBytes(b, v)
		require.NoError(t, err)
		require.Equal(t, buf.Bytes(), b[:n])

		n, err = WireFixed64.Tag(1).WriteValueToBytes(b, v)
		require.NoError(t, err)
		require.Equal(t, buf.Bytes(), b[:n])

		for i := n - 1; i >= 0; i-- {
			n, err = EncodeFixed64ToBytes(b[:i], v)
			require.Error(t, err)
			require.Zero(t, n)

			n, err = WireFixed64.Tag(1).WriteValueToBytes(b[:i], v)
			require.Error(t, err)
			require.Zero(t, n)
		}

		u, n, err = DecodeFixed64FromBytesWithError(buf.Bytes())
		require.NoError(t, err)
		require.Equal(t, v, u)
		require.Equal(t, buf.Len(), n)

		b = buf.Bytes()

		u, err = DecodeFixed64(buf)
		require.NoError(t, err)
		require.Equal(t, v, u)
		require.Zero(t, buf.Len())

		buf = &bytes.Buffer{}
		err = WireFixed64.Tag(1).WriteValue(buf, v)
		require.NoError(t, err)
		require.Equal(t, b, buf.Bytes())

		u, err = WireFixed64.Tag(1).ReadValue(buf)
		require.NoError(t, err)
		require.Equal(t, v, u)
		require.Zero(t, buf.Len())

		for i := n - 1; i > 0; i-- {
			u, n, err = DecodeFixed64FromBytesWithError(b[:i])
			require.Equal(t, io.ErrUnexpectedEOF, err)
			require.Zero(t, n)
			require.Zero(t, u)

			u, n, err = WireFixed64.Tag(1).ReadValueFromBytes(b[:i])
			require.Equal(t, io.ErrUnexpectedEOF, err)
			require.Zero(t, n)
			require.Zero(t, u)

			u, err = WireFixed64.Tag(1).ReadValue(bytes.NewBuffer(b[:i]))
			require.Equal(t, io.EOF, err)
			require.Zero(t, n)
			require.Zero(t, u)

			require.Panics(t, func() { DecodeFixed64FromBytes(b[:i]) })
		}

		if v != 0 {
			break
		}
		v <<= 1
	}
}

func TestEncodeDecodeFixed32(t *testing.T) {
	v := uint32(1)
	for {
		buf := &bytes.Buffer{}
		err := EncodeFixed32(buf, v)
		require.NoError(t, err)

		u, n, err := DecodeFixed32FromBytesWithError(buf.Bytes())
		require.NoError(t, err)
		require.Equal(t, uint64(v), u)
		require.Equal(t, buf.Len(), n)

		b := make([]byte, n)
		n, err = EncodeFixed32ToBytes(b, v)
		require.NoError(t, err)
		require.Equal(t, buf.Bytes(), b[:n])

		n, err = EncodeFixed32ToBytes(b, v)
		require.NoError(t, err)
		require.Equal(t, buf.Bytes(), b[:n])

		n, err = WireFixed32.Tag(1).WriteValueToBytes(b, uint64(v))
		require.NoError(t, err)
		require.Equal(t, buf.Bytes(), b[:n])

		for i := n - 1; i >= 0; i-- {
			n, err = EncodeFixed32ToBytes(b[:i], v)
			require.Error(t, err)
			require.Zero(t, n)

			n, err = WireFixed32.Tag(1).WriteValueToBytes(b[:i], uint64(v))
			require.Error(t, err)
			require.Zero(t, n)
		}

		u, n, err = DecodeFixed32FromBytesWithError(buf.Bytes())
		require.NoError(t, err)
		require.Equal(t, uint64(v), u)
		require.Equal(t, buf.Len(), n)

		b = buf.Bytes()

		u, err = DecodeFixed32(buf)
		require.NoError(t, err)
		require.Equal(t, uint64(v), u)
		require.Zero(t, buf.Len())

		buf = &bytes.Buffer{}
		err = WireFixed32.Tag(1).WriteValue(buf, uint64(v))
		require.NoError(t, err)
		require.Equal(t, b, buf.Bytes())

		u, err = WireFixed32.Tag(1).ReadValue(buf)
		require.NoError(t, err)
		require.Equal(t, uint64(v), u)
		require.Zero(t, buf.Len())

		for i := n - 1; i > 0; i-- {
			u, n, err = DecodeFixed32FromBytesWithError(b[:i])
			require.Equal(t, io.ErrUnexpectedEOF, err)
			require.Zero(t, n)
			require.Zero(t, u)

			u, n, err = WireFixed32.Tag(1).ReadValueFromBytes(b[:i])
			require.Equal(t, io.ErrUnexpectedEOF, err)
			require.Zero(t, n)
			require.Zero(t, u)

			u, err = WireFixed32.Tag(1).ReadValue(bytes.NewBuffer(b[:i]))
			require.Equal(t, io.EOF, err)
			require.Zero(t, n)
			require.Zero(t, u)

			require.Panics(t, func() { DecodeFixed32FromBytes(b[:i]) })
		}

		if v != 0 {
			break
		}
		v <<= 1
	}
}
