// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dropstorage

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func newBufferSlice(page *bufferPage, pageOfs uint32, target []byte) *BufferSlice {
	switch n := len(target); {
	case n == 0:
		panic(throw.IllegalValue())
	case page == nil:
		panic(throw.IllegalValue())
	case n != cap(target):
		target = target[:n:n]
	}
	return &BufferSlice{page, pageOfs, target}
}

var _ io.ReaderFrom = &BufferSlice{}
type BufferSlice struct {
	page    *bufferPage
	pageOfs uint32
	target  []byte
}

func (p *BufferSlice) ReadFrom(r io.Reader) (int64, error) {
	if !p.IsAvailable() {
		panic(throw.IllegalState())
	}
	n, err := io.ReadFull(r, p.target)
	if err == nil && n != len(p.target) {
		err = throw.WithStack(io.ErrShortWrite)
	}
	p.done()
	return int64(n), err
}

func (p *BufferSlice) done() {
	p.target = p.target[:0]
	p.page.wait.Done()
}

func (p *BufferSlice) MustCopyExact(b []byte) {
	switch {
	case !p.IsAvailable():
		panic(throw.IllegalState())
	case len(b) != len(p.target):
		panic(throw.IllegalValue())
	}
	copy(p.target, b)
	p.done()
}

func (p *BufferSlice) MustReadFrom(r io.Reader) {
	if _, err := p.ReadFrom(r); err != nil {
		panic(throw.WithStack(err))
	}
}

func (p *BufferSlice) PageNo() uint32 {
	return p.page.pageNo
}

func (p *BufferSlice) SliceOfs() uint32 {
	return p.pageOfs
}

func (p *BufferSlice) SliceLen() uint32 {
	return uint32(cap(p.target))
}

func (p *BufferSlice) IsAvailable() bool {
	return len(p.target) != 0
}

