// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package longbits

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFoldUint64(t *testing.T) {
	require.Zero(t, FoldUint64(0))

	require.Equal(t, uint32(2), FoldUint64(2))

	require.Equal(t, uint32(math.MaxUint32), FoldUint64(math.MaxUint32))

	require.Equal(t, uint32(1), FoldUint64(math.MaxUint32+1))

	require.Zero(t, FoldUint64(math.MaxUint64))
}

func TestEqualFixedLenWriterTo(t *testing.T) {
	require.False(t, Equal(nil, nil))

	bits1 := NewBits64(0)
	require.False(t, Equal(&bits1, nil))

	require.False(t, Equal(nil, &bits1))

	bits2 := NewBits64(0)
	require.True(t, Equal(&bits1, &bits2))

	bits2 = NewBits64(1)
	require.False(t, Equal(&bits1, &bits2))
}

func TestCompare(t *testing.T) {
	require.False(t, (&writerToComparer{}).compare(nil, nil))

	bits1 := NewBits64(0)
	require.False(t, (&writerToComparer{}).compare(&bits1, nil))

	require.False(t, (&writerToComparer{}).compare(nil, &bits1))

	bits2 := NewBits64(0)
	require.True(t, (&writerToComparer{}).compare(&bits1, &bits2))

	bits3 := NewBits128(0, 0)
	require.False(t, (&writerToComparer{}).compare(&bits1, &bits3))

	bits1 = NewBits64(1)
	require.False(t, (&writerToComparer{}).compare(&bits1, &bits2))
}

func TestWrite(t *testing.T) {
	require.Panics(t, func() { _, _ = (&writerToComparer{}).Write(nil) })

	bits := NewBits64(0)
	fr := NewFixedReaderMock(t)
	fr.WriteToMock.Set(func(io.Writer) (int64, error) { return 0, errors.New("test") })
	n, err := (&writerToComparer{other: fr}).Write(bits.AsBytes())
	require.NotEqual(t, nil, err)

	require.Zero(t, n)

	n, err = (&writerToComparer{other: &bits}).Write(bits.AsBytes())
	require.Nil(t, err)

	require.Equal(t, 8, n)
}

func TestAsByteString(t *testing.T) {
	fs := fixedSize{}
	require.Empty(t, fs.AsByteString())

	fs = fixedSize{data: []byte{'a', 'b', 'c'}}
	require.Equal(t, ByteString("abc"), fs.AsByteString())
}

func TestWriteTo(t *testing.T) {
	fs := fixedSize{data: []byte{0}}
	buf := &bytes.Buffer{}
	n, err := fs.WriteTo(buf)
	require.Nil(t, err)

	require.Equal(t, int64(1), n)
}

func TestRead(t *testing.T) {
	item := byte(3)
	fs := fixedSize{data: []byte{item}}
	buf := make([]byte, 2)
	n := fs.CopyTo(buf)
	require.Equal(t, 1, n)
	require.Equal(t, item, buf[0])
}

func TestFoldToUint64(t *testing.T) {
	require.Equal(t, uint64(0), fixedSize{data: []byte{}}.FoldToUint64())
	require.Equal(t, uint64(0x01), fixedSize{data: []byte{1}}.FoldToUint64())
	require.Equal(t, uint64(0x0201), fixedSize{data: []byte{1, 2}}.FoldToUint64())

	fs := fixedSize{data: []byte{1, 2, 3, 4, 5, 6, 7, 8}}
	require.Equal(t, uint64(0x0807060504030201), fs.FoldToUint64())
	fs.data = append(fs.data, 9)
	require.Equal(t, uint64(0x0807060504030208), fs.FoldToUint64())
	fs.data = append(fs.data, 10)
	require.Equal(t, uint64(0x0807060504030808), fs.FoldToUint64())
	fs.data = append(fs.data, 11, 12, 13, 14, 15, 16)
	require.Equal(t, uint64(0x1808080808080808), fs.FoldToUint64())
	fs.data = append(fs.data, 0x10, 0x20, 0x30, 0x40, 0x50, 0x60, 0x70, 0x80)
	require.Equal(t, uint64(0x9878685848382818), fs.FoldToUint64())
}

func TestCutOutUint64(t *testing.T) {
	require.Equal(t, uint64(0), fixedSize{data: []byte{}}.CutOutUint64())
	require.Equal(t, uint64(0x01), fixedSize{data: []byte{1}}.CutOutUint64())
	require.Equal(t, uint64(0x0201), fixedSize{data: []byte{1, 2}}.CutOutUint64())
	require.Equal(t, uint64(0x807060504030201), fixedSize{data: []byte{1, 2, 3, 4, 5, 6, 7, 8}}.CutOutUint64())
}

func TestFixedByteSize(t *testing.T) {
	fs := fixedSize{data: []byte{1, 2}}
	require.Equal(t, len(fs.data), fs.FixedByteSize())
}

func TestCopyTo(t *testing.T) {
	fs := fixedSize{data: []byte{1, 2}}
	d := []byte{0, 0, 0}

	require.Equal(t, 1, fs.CopyTo(d[:1]))
	require.Equal(t, fs.data[0], d[0])
	require.Zero(t, d[1])

	require.Equal(t, len(fs.data), fs.CopyTo(d))
	require.Equal(t, fs.data, d[:len(fs.data)])
}

func TestNewFixedReader(t *testing.T) {
	data := []byte{1, 2, 3}
	fr := WrapBytes(data)
	require.Len(t, AsBytes(fr), len(data))

	require.Equal(t, data[1], AsBytes(fr)[1])
}

func TestCopyFixedSize(t *testing.T) {
	var b [1024/8]byte
	_, _ = rand.Read(b[:])

	for i := len(b); i >= 0; i-- {
		orig := WrapBytes(b[:i])
		cp := CopyFixed(orig)

		require.Equal(t, i, cp.FixedByteSize(), i)
		require.True(t, Equal(orig, cp), i)

		switch i*8 {
		case 0:
			require.Equal(t, EmptyByteString, cp, i)
			require.Nil(t, AsBytes(cp), i)
			continue
		case 64:
			require.IsType(t, Bits64{}, cp, i)
		case 128:
			require.IsType(t, Bits128{}, cp, i)
		case 224:
			require.IsType(t, Bits224{}, cp, i)
		case 256:
			require.IsType(t, Bits256{}, cp, i)
		case 512:
			require.IsType(t, Bits512{}, cp, i)
		default:
			require.IsType(t, fixedSize{}, cp, i)
		}
		require.Equal(t, b[:i], AsBytes(cp), i)
	}
}

func TestEqual(t *testing.T) {
	empty0 := WrapBytes([]byte{})
	empty1 := WrapBytes([]byte{})
	require.False(t, Equal(nil, nil))
	require.False(t, Equal(nil, empty1))
	require.False(t, Equal(empty0, nil))
	require.True(t, Equal(empty0, empty1))
	require.False(t, Equal(empty0, WrapBytes([]byte{1})))
	require.True(t, Equal(WrapBytes([]byte{1}), WrapBytes([]byte{1})))
}

func TestEqualToBytes(t *testing.T) {
	empty0 := WrapBytes([]byte{})
	empty1 := make([]byte, 0)
	require.False(t, EqualToBytes(nil, nil))
	require.False(t, EqualToBytes(nil, empty1))
	require.True(t, EqualToBytes(empty0, nil))
	require.True(t, EqualToBytes(empty0, empty1))
	require.False(t, EqualToBytes(empty0, []byte{1}))
	require.True(t, EqualToBytes(WrapBytes([]byte{1}), []byte{1}))
}
