// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package longbits

import (
	"bytes"
	"errors"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type Foldable interface {
	FoldToUint64() uint64
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits.FixedReader -o . -s _mock.go -g
type FixedReader interface {
	io.WriterTo
	CopyTo(p []byte) int
	AsByteString() ByteString

	FixedByteSize() int
}

//go:generate minimock -i github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits.FoldableReader -o . -s _mock.go -g
type FoldableReader interface {
	FixedReader
	Foldable
}

func FoldUint64(v uint64) uint32 {
	return uint32(v) ^ uint32(v>>32)
}

func Equal(t, o FixedReader) bool {
	switch {
	case t == nil || o == nil:
		return false
	case t.FixedByteSize() == 0:
		return o.FixedByteSize() == 0
	}
	return (&writerToComparer{}).compare(t, o)
}

func EqualToBytes(t FixedReader, o []byte) bool {
	switch {
	case t == nil:
		return false
	case t.FixedByteSize() == 0:
		return len(o) == 0
	case len(o) == 0:
		return false
	}
	return (&writerToComparer{}).compareBytes(o, t)
}

type writerToComparer struct {
	thisValue *[]byte
	other     io.WriterTo
	result    bool
}

func (c *writerToComparer) compareBytes(this []byte, other FixedReader) bool {
	if other == nil || len(this) != other.FixedByteSize() {
		return false
	}
	c.thisValue = &this
	c.other = other
	_, _ = other.WriteTo(c)
	return c.other == nil && c.result
}

func (c *writerToComparer) compare(this, other FixedReader) bool {
	c.thisValue = nil
	if this == nil || other == nil || this.FixedByteSize() != other.FixedByteSize() {
		return false
	}
	c.other = other
	_, _ = this.WriteTo(c)
	return c.other == nil && c.result
}

func (c *writerToComparer) Write(otherValue []byte) (int, error) {
	if c.other == nil {
		panic("content of FixedReader must be read/written all at once")
	}
	if c.thisValue == nil {
		c.thisValue = &otherValue
		_, err := c.other.WriteTo(c)
		if err != nil {
			return 0, err
		}
	} else {
		c.other = nil // mark "done"
		c.result = bytes.Equal(*c.thisValue, otherValue)
	}
	return len(otherValue), nil
}

type fixedSize struct {
	data []byte
}

func (c fixedSize) AsByteString() ByteString {
	return ByteString(c.data)
}

func (c fixedSize) String() string {
	return ByteString(c.data).String()
}

func (c fixedSize) WriteTo(w io.Writer) (n int64, err error) {
	n32, err := w.Write(c.data)
	return int64(n32), err
}

func (c fixedSize) CopyTo(p []byte) (n int) {
	return copy(p, c.data)
}

func (c fixedSize) FoldToUint64() uint64 {
	return FoldToUint64(c.data)
}

func (c fixedSize) CutOutUint64() uint64 {
	return CutOutUint64(c.data)
}

func (c fixedSize) FixedByteSize() int {
	return len(c.data)
}

func AsBytes(v FixedReader) []byte {
	n := v.FixedByteSize()
	if n == 0 {
		return nil
	}
	data := make([]byte, n)
	if v.CopyTo(data) != len(data) {
		panic(throw.Impossible())
	}
	return data
}

func NewMutableFixedSize(data []byte) FoldableReader {
	return fixedSize{data}
}

func CopyToMutable(v FixedReader) FoldableReader {
	return fixedSize{AsBytes(v)}
}

func NewImmutableFixedSize(data []byte) FoldableReader {
	return CopyBytes(data).AsReader()
}

func CopyToImmutable(v FixedReader) FoldableReader {
	return CopyBytes(AsBytes(v)).AsReader()
}

func CopyFixedSize(v FixedReader) FoldableReader {
	return fixedSize{AsBytes(v)}
}

func CopyAllBytes(to []byte, from FixedReader) error {
	if n := from.FixedByteSize(); n != len(to) {
		if n < len(to) {
			return io.ErrShortBuffer
		}
		return io.ErrShortWrite
	}
	from.CopyTo(to)
	return nil
}

func VerifyReadAt(b []byte, off int64, max int) (n int, err error) {
	switch {
	case off < 0:
		return 0, errors.New("negative offset")
	case off > int64(max):
		return 0, io.ErrUnexpectedEOF
	case len(b) == 0:
		return 0, nil
	case max == int(off):
		return 0, io.EOF
	}
	return len(b), nil
}
