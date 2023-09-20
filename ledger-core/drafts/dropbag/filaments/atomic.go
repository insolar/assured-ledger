package filaments

import (
	"sync/atomic"
	"unsafe"
)

func NewAtomicEntry(entry *WriteEntry) AtomicEntry {
	return AtomicEntry{entry}
}

type AtomicEntry struct {
	entry *WriteEntry
}

func (p *AtomicEntry) _ptr() *unsafe.Pointer {
	return (*unsafe.Pointer)((unsafe.Pointer)(&p.entry))
}

func (p *AtomicEntry) Get() *WriteEntry {
	return (*WriteEntry)(atomic.LoadPointer(p._ptr()))
}

func (p *AtomicEntry) Set(v *WriteEntry) {
	atomic.StorePointer(p._ptr(), (unsafe.Pointer)(v))
}

func (p *AtomicEntry) Swap(v *WriteEntry) *WriteEntry {
	return (*WriteEntry)(atomic.SwapPointer(p._ptr(), (unsafe.Pointer)(v)))
}

func (p *AtomicEntry) CmpAndSwap(expected, new *WriteEntry) bool {
	return atomic.CompareAndSwapPointer(p._ptr(), (unsafe.Pointer)(expected), (unsafe.Pointer)(new))
}
