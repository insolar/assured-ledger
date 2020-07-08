// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package dropstorage

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type PagedBuffer struct {
	pageBytes uint32
	mutex    sync.RWMutex
	current  *bufferPage
	earliest *bufferPage
}

func (p *PagedBuffer) getCurrentPage() *bufferPage {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.current
}

func (p *PagedBuffer) allocatePage(ln int) (*bufferPage, uint32, []byte) {
	if ln <= 0 {
		panic(throw.IllegalValue())
	}

	allocateLen := uint32(ln)

	if allocateLen > p.pageBytes {
		return p.allocateLargePage(allocateLen)
	}

	cur := p.getCurrentPage()
	if cur == nil {
		cur = p.allocateNewPage(nil)
	}

	for {
		if ofs, buf := cur.allocateWrite(allocateLen); buf != nil {
			cur.wait.Done()
			return cur, ofs, buf
		}
		cur = p.allocateNewPage(cur)
	}

}

func (p *PagedBuffer) allocateNewPage(old *bufferPage) *bufferPage {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.current != old {
		return p.current
	}
	return p._connectPage(newBufferPage(p.pageBytes))
}

func (p *PagedBuffer) _connectPage(new *bufferPage) *bufferPage {
	old := p.current
	if old != nil {
		old.next = new
		new.pageNo = old.pageNo + 1
	} else {
		p.earliest = new
	}
	p.current = new
	return new
}

func (p *PagedBuffer) allocateLargePage(ln uint32) (*bufferPage, uint32, []byte) {
	page := newBufferPage(ln)
	ofs, b := page.allocateWrite(ln)
	if b == nil {
		panic(throw.Impossible())
	}
	defer page.wait.Done()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	cur := p.current
	if cur != nil {
		p._connectPage(page)
	}
	p._connectPage(newBufferPage(p.pageBytes))
	return page, ofs, b
}

func (p *PagedBuffer) AllocateWrite(ln int) *BufferSlice {
	pg, ofs, b := p.allocatePage(ln)
	return newBufferSlice(pg, ofs, b)
}

func (p *PagedBuffer) flushPages() *bufferPage {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	cur := p.current
	if cur == nil || cur.used.Load() == 0 {
		return nil
	}
	first := p.earliest
	if first == nil {
		panic(throw.Impossible())
	}

	pg := newFlushedPage(cur.pageNo + 1)
	p.current = pg
	p.earliest = pg
	return first
}

func (p *PagedBuffer) FlushPages() *FlushedPages {
	first := p.flushPages()
	if first == nil {
		return nil
	}
	return &FlushedPages{ first:first }
}

func newFlushedPage(pageNo uint32) *bufferPage {
	return &bufferPage{ pageNo:pageNo }
}

func newBufferPage(capacity uint32) *bufferPage {
	return &bufferPage{ buffer: make([]byte, capacity) }
}

type bufferPage struct {
	pageNo uint32
	used   atomickit.Uint32
	wait   sync.WaitGroup
	next   *bufferPage
	buffer []byte
}

func (p *bufferPage) allocateWrite(ln uint32) (ofs uint32, b []byte) {
	for {
		used := p.used.Load()
		next := used + ln
		switch {
		case next > uint32(cap(p.buffer)):
			return 0, nil
		case p.used.CompareAndSwap(used, next):
			p.wait.Add(2) // +1 for allocation cycle, +1 for writer
			return used, p.buffer[used:next]
		}
	}
}

