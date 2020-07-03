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
		p.allocateLargePage(allocateLen)
	}

	cur := p.getCurrentPage()
	if cur == nil {
		cur = p.allocateNewPage(nil)
	}

	for {
		if ofs, buf := cur.allocateWrite(allocateLen); buf != nil {
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
	new.prev = old
	if old != nil {
		new.pageNo = old.pageNo + 1
		new.absPos = old.absPos + uint64(old.used.Load())
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
	return newBufferSlice(pg.pageNo, ofs, b)
}

func (p *PagedBuffer) _flushPages() *bufferPage {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.current == nil || p.current.used.Load() == 0 {
		return nil
	}


}

func (p *PagedBuffer) FlushPages() {
	p.mutex.Lock()
	defer p.mutex.Unlock()


}

func newFlushedPage(capacity uint32) *bufferPage {
	return &bufferPage{ buffer: make([]byte, capacity, capacity) }
}

func newBufferPage(capacity uint32) *bufferPage {
	return &bufferPage{ buffer: make([]byte, capacity, capacity) }
}

type bufferPage struct {
	pageNo uint32
	used   atomickit.Uint32
	absPos uint64
	prev   *bufferPage
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
			return used, p.buffer[used:next]
		}
	}
}

