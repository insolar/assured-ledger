package smachine

import (
	"sync"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// SlotPool by default recycles deallocated pages to mitigate possible memory leak through SlotLink references
// When flow of slots varies a lot and there is no long-living links then deallocateOnCleanup can be enabled.
func newSlotPool(pageSize uint16, deallocateOnCleanup bool) SlotPool {
	if pageSize < 1 {
		panic("illegal value")
	}
	return SlotPool{
		slotPages:  []slotPage{make(slotPage, pageSize)},
		deallocate: deallocateOnCleanup,
	}
}

type slotPage = []Slot

type SlotPool struct {
	mutex sync.RWMutex

	slotPages []slotPage // LOCK specifics. This slice has mixed access - see ScanAndCleanup

	unusedSlots SlotQueue
	emptyPages  []slotPage
	slotPgPos   uint16
	deallocate  bool
	blockAlloc  bool
}

func (p *SlotPool) initSlotPool() {
	if p.slotPages == nil {
		panic("illegal nil")
	}
	p.unusedSlots.initSlotQueue(UnusedSlots)
}

func (p *SlotPool) Count() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.count()
}

func (p *SlotPool) count() int {
	n := len(p.slotPages)
	if n == 0 {
		return 0
	}
	return (n-1)*len(p.slotPages[0]) + int(p.slotPgPos) - p.unusedSlots.Count()
}

func (p *SlotPool) Capacity() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	n := len(p.slotPages)
	if n == 0 {
		return 0
	}
	return len(p.slotPages) * len(p.slotPages[0])
}

func (p *SlotPool) IsEmpty() bool {
	return p.Count() == 0
}

func (p *SlotPool) BlockIfEmpty(stopFn func() bool) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	switch {
	case p.blockAlloc:
		return false
	case p.count() != 0:
		return false
	case stopFn != nil && !stopFn():
		return false
	}
	p.blockAlloc = true
	return true
}

// AllocateSlot creates or reuses a slot. The returned slot is marked BUSY and is at step 0.
// Work of AllocateSlot is not blocked during ScanAndCleanup
func (p *SlotPool) AllocateSlot(m *SlotMachine, id SlotID) (slot *Slot) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	switch {
	case p.blockAlloc:
		return nil
	case !p.unusedSlots.IsEmpty():
		slot = p.unusedSlots.First()
		slot.removeFromQueue()
		m.ensureLocal(slot)
	case p.slotPages == nil:
		panic(throw.IllegalState())
	default:
		lenSlots := len(p.slotPages[0])
		if int(p.slotPgPos) == lenSlots {
			p.slotPages = append(p.slotPages, p.slotPages[0])
			p.slotPages[0] = p.allocatePage(lenSlots)
			p.slotPgPos = 0
		}
		slot = &p.slotPages[0][p.slotPgPos]
		slot.machine = m
		p.slotPgPos++
	}
	slot._slotAllocated(id)

	return slot
}

// RecycleSlot adds a slot to unused slot queue.
// Work of RecycleSlot is not blocked during ScanAndCleanup
func (p *SlotPool) RecycleSlot(slot *Slot) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.unusedSlots.AddFirst(slot)
}

type SlotPageScanFunc func([]Slot) (isPageEmptyOrWeak, hasWeakSlots bool)
type SlotDisposeFunc func(*Slot)

// ScanAndCleanup is safe and non-blocking for parallel calls of AllocateSlot(),
// but ScanAndCleanup can't be called multiple times in parallel
func (p *SlotPool) ScanAndCleanup(cleanupWeak bool, disposeWeakFn SlotDisposeFunc, scanPageFn SlotPageScanFunc) bool {
	// ScanAndCleanup doesn't need to hold a lock, because AllocateSlot either appends or change [0].
	// And here we get [0] as an explicit slice (partialPage) while other pages as a slice of pages.
	partialPage, fullPages := p.getScanPages()

	isAllEmptyOrWeak, hasSomeWeakSlots := scanPageFn(partialPage)

	firstNil := 0
	for i, slotPage := range fullPages {
		isPageEmptyOrWeak, hasWeakSlots := scanPageFn(slotPage)
		switch {
		case !isPageEmptyOrWeak:
			isAllEmptyOrWeak = false
		case hasWeakSlots:
			hasSomeWeakSlots = true
		case firstNil == 0:
			firstNil = i + 1
			fallthrough
		default:
			p.recycleEmptyPage(i + 1)
		}
	}

	cleanupAll := isAllEmptyOrWeak && (cleanupWeak || !hasSomeWeakSlots)
	return p.cleanup(len(partialPage), len(fullPages), firstNil, cleanupAll, disposeWeakFn)
}

func (p *SlotPool) getScanPages() (partial slotPage, full []slotPage) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	switch n := len(p.slotPages); {
	case n > 1:
		return p.slotPages[0][:p.slotPgPos], p.slotPages[1:]
	case n == 0 || p.slotPgPos == 0:
		return nil, nil
	default:
		return p.slotPages[0][:p.slotPgPos], nil
	}
}

func (p *SlotPool) cleanup(partialCount, fullCount, firstNil int, cleanupAll bool, disposeWeakFn SlotDisposeFunc) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if cleanupAll && fullCount+1 == len(p.slotPages) && p.slotPgPos == uint16(partialCount) {
		// As AllocateSlot() can run in parallel, there can be new slots and slot pages
		// so, full cleanup can only be applied when there were no additions
		for i, slotPage := range p.slotPages {
			if slotPage == nil {
				break
			}
			for j := range slotPage {
				slot := &slotPage[j]
				if !slot.isEmpty() {
					disposeWeakFn(slot)
				}
				switch qt := slot.QueueType(); {
				case qt == UnusedSlots:
					slot.removeFromQueue()
				case i != 0:
					panic(throw.IllegalState())
				case qt != NoQueue:
					panic(throw.IllegalState())
				}
			}
		}
		if p.unusedSlots.Count() != 0 {
			panic(throw.IllegalState())
		}
		p.slotPages = p.slotPages[:1]
		p.slotPgPos = 0
		return true
	}

	if firstNil > 0 {
		j := firstNil
		if p.slotPages[j] != nil {
			panic(throw.IllegalState())
		}

		for i := j + 1; i < len(p.slotPages); i++ {
			if sp := p.slotPages[i]; sp != nil {
				p.slotPages[j] = sp
				j++
			}
		}
		p.slotPages = p.slotPages[:j]
	}
	return false
}

func (p *SlotPool) allocatePage(lenSlots int) []Slot {
	for n := len(p.emptyPages) - 1; n >= 0; n-- {
		pg := p.emptyPages[n]
		p.emptyPages[n] = nil
		p.emptyPages = p.emptyPages[:n]

		if len(pg) == lenSlots {
			return pg
		}
	}
	return make([]Slot, lenSlots)
}

func (p *SlotPool) recycleEmptyPage(pageNo int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	page := p.slotPages[pageNo]
	switch {
	case pageNo == 0:
		panic(throw.IllegalState())
	case page == nil:
		panic(throw.IllegalState())
	}
	p.slotPages[pageNo] = nil

	for i := range page {
		slot := &page[i]
		if slot.QueueType() != UnusedSlots {
			panic(throw.IllegalState())
		}
		slot.removeFromQueue()
	}

	if !p.deallocate {
		p.emptyPages = append(p.emptyPages, page)
		return
	}

	for i := range page {
		if !page[i].unsetMachine() {
			break
		}
	}
}

// cleanupEmpty can ONLY be applied after cleanup of regular pages
func (p *SlotPool) cleanupEmpty() {
	if !p.IsEmpty() {
		panic(throw.IllegalState())
	}
	p.unusedSlots.RemoveAll()
	p.slotPages = nil

	pages := p.emptyPages
	p.emptyPages = nil

	for _, page := range pages {
		for i := range page {
			if !page[i].unsetMachine() {
				break
			}
		}
	}
}
