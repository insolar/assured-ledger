package smsync

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
)

func w(a ...int) []int {
	return a
}

func fakeSlotLink(id smachine.SlotID) (r smachine.SlotLink) {
	type slot struct {
		idAndStep uint64
	}
	type slotLink struct {
		id smachine.SlotID
		s  *smachine.Slot
	}

	// DO NOT do this never ever

	s := &smachine.Slot{}
	sp := (*slot)(unsafe.Pointer(s))
	sp.idAndStep = uint64(id) | 0x1<<40 // magic!

	rp := (*slotLink)(unsafe.Pointer(&r))
	rp.id = id
	rp.s = s
	return
}

func TestSemaHierarchy(t *testing.T) {
	master := newSemaphore(2, true, "master", 0)

	child0 := newSemaphoreChild(master, AllowPartialRelease, 1, false, "child0")
	child1 := newSemaphoreChild(master, AllowPartialRelease, 2, false, "child1")
	child2 := newSemaphoreChild(master, 0, 2, false, "child2")

	require.Equal(t, []int{0, 0}, w(master.GetCounts()))
	require.Equal(t, []int{0, 0}, w(child0.GetCounts()))
	require.Equal(t, []int{0, 0}, w(child1.GetCounts()))
	require.Equal(t, []int{0, 0}, w(child2.GetCounts()))

	/*** child0 ***/

	ok, dep01 := child0.CreateDependency(fakeSlotLink(1), 0)
	require.NotNil(t, dep01)
	require.True(t, ok.IsPassed())
	require.Equal(t, []int{1, 0}, w(master.GetCounts()))
	require.Equal(t, []int{1, 0}, w(child0.GetCounts()))

	ok, dep02 := child0.CreateDependency(fakeSlotLink(2), 0)
	require.NotNil(t, dep02)
	require.False(t, ok.IsPassed())
	require.Equal(t, []int{1, 0}, w(master.GetCounts()))
	require.Equal(t, []int{1, 1}, w(child0.GetCounts()))

	ok, dep03 := child0.CreateDependency(fakeSlotLink(3), 0)
	require.NotNil(t, dep03)
	require.False(t, ok.IsPassed())
	require.Equal(t, []int{1, 0}, w(master.GetCounts()))
	require.Equal(t, []int{1, 2}, w(child0.GetCounts()))

	/*** child1 ***/

	ok, dep14 := child1.CreateDependency(fakeSlotLink(4), 0)
	require.NotNil(t, dep14)
	require.True(t, ok.IsPassed())
	require.Equal(t, []int{2, 0}, w(master.GetCounts()))
	require.Equal(t, []int{1, 0}, w(child1.GetCounts()))

	ok, dep15 := child1.CreateDependency(fakeSlotLink(5), 0)
	require.NotNil(t, dep15)
	require.False(t, ok.IsPassed())
	require.Equal(t, []int{2, 1}, w(master.GetCounts()))
	require.Equal(t, []int{2, 0}, w(child1.GetCounts()))

	ok, dep16 := child1.CreateDependency(fakeSlotLink(6), 0)
	require.NotNil(t, dep16)
	require.False(t, ok.IsPassed())
	require.Equal(t, []int{2, 1}, w(master.GetCounts()))
	require.Equal(t, []int{2, 1}, w(child1.GetCounts()))

	/*** child2 ***/

	ok, dep27 := child2.CreateDependency(fakeSlotLink(7), 0)
	require.NotNil(t, dep27)
	require.False(t, ok.IsPassed())
	require.Equal(t, []int{2, 2}, w(master.GetCounts()))
	require.Equal(t, []int{1, 0}, w(child2.GetCounts()))

	ok, dep28 := child2.CreateDependency(fakeSlotLink(8), 0)
	require.NotNil(t, dep28)
	require.False(t, ok.IsPassed())
	require.Equal(t, []int{2, 3}, w(master.GetCounts()))
	require.Equal(t, []int{2, 0}, w(child2.GetCounts()))

	ok, dep29 := child2.CreateDependency(fakeSlotLink(9), 0)
	require.NotNil(t, dep29)
	require.False(t, ok.IsPassed())
	require.Equal(t, []int{2, 3}, w(master.GetCounts()))
	require.Equal(t, []int{2, 1}, w(child2.GetCounts()))

	/*** Partial release ***/

	require.Equal(t, smachine.NotPassed, child1.UseDependency(dep15, 0))
	require.Equal(t, smachine.NotPassed, master.UseDependency(dep15, 0))
	rd, dp, _, sl := master.ReleaseDependency(dep15)
	require.True(t, rd)
	require.True(t, dp == dep15)
	require.Equal(t, []int{2, 2}, w(master.GetCounts()))
	require.Equal(t, []int{2, 1}, w(child1.GetCounts()))
	require.Equal(t, 0, len(sl))
	// This is a special case - as dependency has explicitly released parent, the it can only acquire the parent
	require.Equal(t, smachine.Impossible, child1.UseDependency(dep15, 0))

	require.Equal(t, smachine.NotPassed, child1.UseDependency(dep15, smachine.SyncIgnoreFlags))
	require.Equal(t, smachine.NotPassed, master.UseDependency(dep15, smachine.SyncIgnoreFlags))
	require.Equal(t, []int{2, 2}, w(master.GetCounts()))
	require.Equal(t, []int{2, 1}, w(child1.GetCounts()))

	require.Equal(t, smachine.Passed, child1.UseDependency(dep14, 0))
	require.Equal(t, smachine.Passed, master.UseDependency(dep14, 0))
	rd, dp, _, sl = master.ReleaseDependency(dep14)
	require.True(t, rd)
	require.True(t, dp == dep14)
	require.Equal(t, []int{2, 1}, w(master.GetCounts()))
	require.Equal(t, []int{2, 1}, w(child1.GetCounts()))
	// This is a special case - as dependency has explicitly released parent, the it can only acquire the parent
	require.Equal(t, smachine.Impossible, child1.UseDependency(dep14, 0))

	require.Equal(t, smachine.NotPassed, child1.UseDependency(dep14, smachine.SyncIgnoreFlags))
	require.Equal(t, smachine.NotPassed, master.UseDependency(dep14, smachine.SyncIgnoreFlags))
	require.Equal(t, []int{2, 1}, w(master.GetCounts()))
	require.Equal(t, []int{2, 1}, w(child1.GetCounts()))

	require.Equal(t, 1, len(sl))
	require.Equal(t, smachine.SlotID(7), sl[0].SlotID())

	/*** Partial release not allowed ***/

	require.Equal(t, smachine.NotPassed, master.UseDependency(dep29, 0))
	rd, dp, _, sl = master.ReleaseDependency(dep29)
	require.False(t, rd)
	require.True(t, dp == dep29)
	require.Equal(t, 0, len(sl))

	rd, dp, _, sl = child2.ReleaseDependency(dep29)
	require.True(t, rd)
	require.Nil(t, dp)
	require.Equal(t, []int{2, 1}, w(master.GetCounts()))
	require.Equal(t, []int{2, 0}, w(child2.GetCounts()))
	require.Equal(t, 0, len(sl))

	require.Equal(t, smachine.Passed, master.UseDependency(dep27, 0))
	rd, dp, _, sl = master.ReleaseDependency(dep27)
	require.False(t, rd)
	require.True(t, dp == dep27)
	require.Equal(t, 0, len(sl))

	rd, dp, _, sl = child2.ReleaseDependency(dep27)
	require.True(t, rd)
	require.Nil(t, dp)
	require.Equal(t, []int{2, 0}, w(master.GetCounts()))
	require.Equal(t, []int{1, 0}, w(child2.GetCounts()))
	require.Equal(t, 1, len(sl))
	require.Equal(t, smachine.SlotID(8), sl[0].SlotID())

	/*** Clear out child2 ***/

	_, sl = dep28.ReleaseAll()
	require.Equal(t, []int{1, 0}, w(master.GetCounts()))
	require.Equal(t, []int{0, 0}, w(child2.GetCounts()))
	require.Equal(t, 0, len(sl))

	/*** reacquire parent ***/

	require.Equal(t, smachine.NotPassed, master.UseDependency(dep16, 0))
	require.Equal(t, []int{1, 0}, w(master.GetCounts()))
	require.Equal(t, []int{2, 1}, w(child1.GetCounts()))

	// reacquire the parent
	require.Equal(t, smachine.Passed, master.UseDependency(dep14, 0))
	require.Equal(t, []int{2, 0}, w(master.GetCounts()))
	require.Equal(t, []int{2, 1}, w(child1.GetCounts()))

	// reacquire the parent
	require.Equal(t, smachine.NotPassed, master.UseDependency(dep15, 0))
	require.Equal(t, []int{2, 1}, w(master.GetCounts()))
	require.Equal(t, []int{2, 1}, w(child1.GetCounts()))

	/*** clear out child1 ***/

	rd, dp, _, sl = child2.ReleaseDependency(dep14)
	require.True(t, rd)
	require.Nil(t, dp)
	require.Equal(t, []int{2, 1}, w(master.GetCounts()))
	require.Equal(t, []int{2, 0}, w(child1.GetCounts()))
	require.Equal(t, 1, len(sl))
	require.Equal(t, smachine.SlotID(5), sl[0].SlotID())

	_, sl = dep15.ReleaseAll()
	require.Equal(t, []int{2, 0}, w(master.GetCounts()))
	require.Equal(t, []int{1, 0}, w(child1.GetCounts()))
	require.Equal(t, 1, len(sl))
	require.Equal(t, smachine.SlotID(6), sl[0].SlotID())

	_, sl = dep16.ReleaseAll()
	require.Equal(t, []int{1, 0}, w(master.GetCounts()))
	require.Equal(t, []int{0, 0}, w(child1.GetCounts()))
	require.Equal(t, 0, len(sl))

	/*** clear out child0 ***/

	_, sl = dep03.ReleaseAll()
	require.Equal(t, []int{1, 0}, w(master.GetCounts()))
	require.Equal(t, []int{1, 1}, w(child0.GetCounts()))
	require.Equal(t, 0, len(sl))

	_, sl = dep02.ReleaseAll()
	require.Equal(t, []int{1, 0}, w(master.GetCounts()))
	require.Equal(t, []int{1, 0}, w(child0.GetCounts()))
	require.Equal(t, 0, len(sl))

	_, sl = dep01.ReleaseAll()
	require.Equal(t, []int{0, 0}, w(master.GetCounts()))
	require.Equal(t, []int{0, 0}, w(child0.GetCounts()))
	require.Equal(t, 0, len(sl))
}

func TestSemaHierarchyAdjustments(t *testing.T) {
	master := newSemaphore(2, true, "master", 0)

	child0 := newSemaphoreChild(master, AllowPartialRelease, 1, true, "child0")
	child1 := newSemaphoreChild(master, AllowPartialRelease, 2, true, "child1")

	require.Equal(t, []int{0, 0}, w(master.GetCounts()))
	require.Equal(t, []int{0, 0}, w(child0.GetCounts()))
	require.Equal(t, []int{0, 0}, w(child1.GetCounts()))

	/*** child0 ***/

	ok, dep01 := child0.CreateDependency(fakeSlotLink(1), 0)
	require.NotNil(t, dep01)
	require.True(t, ok.IsPassed())
	require.Equal(t, []int{1, 0}, w(master.GetCounts()))
	require.Equal(t, []int{1, 0}, w(child0.GetCounts()))

	ok, dep02 := child0.CreateDependency(fakeSlotLink(2), 0)
	require.NotNil(t, dep02)
	require.False(t, ok.IsPassed())
	require.Equal(t, []int{1, 0}, w(master.GetCounts()))
	require.Equal(t, []int{1, 1}, w(child0.GetCounts()))

	ok, dep03 := child0.CreateDependency(fakeSlotLink(3), 0)
	require.NotNil(t, dep03)
	require.False(t, ok.IsPassed())
	require.Equal(t, []int{1, 0}, w(master.GetCounts()))
	require.Equal(t, []int{1, 2}, w(child0.GetCounts()))

	/*** child1 ***/

	ok, dep14 := child1.CreateDependency(fakeSlotLink(4), 0)
	require.NotNil(t, dep14)
	require.True(t, ok.IsPassed())
	require.Equal(t, []int{2, 0}, w(master.GetCounts()))
	require.Equal(t, []int{1, 0}, w(child1.GetCounts()))

	ok, dep15 := child1.CreateDependency(fakeSlotLink(5), 0)
	require.NotNil(t, dep15)
	require.False(t, ok.IsPassed())
	require.Equal(t, []int{2, 1}, w(master.GetCounts()))
	require.Equal(t, []int{2, 0}, w(child1.GetCounts()))

	ok, dep16 := child1.CreateDependency(fakeSlotLink(6), 0)
	require.NotNil(t, dep16)
	require.False(t, ok.IsPassed())
	require.Equal(t, []int{2, 1}, w(master.GetCounts()))
	require.Equal(t, []int{2, 1}, w(child1.GetCounts()))

	/*** Partial release ***/

	require.Equal(t, smachine.NotPassed, child1.UseDependency(dep15, 0))
	require.Equal(t, smachine.NotPassed, master.UseDependency(dep15, 0))
	rd, dp, _, sl := master.ReleaseDependency(dep15)
	require.True(t, rd)
	require.True(t, dp == dep15)
	require.Equal(t, []int{2, 0}, w(master.GetCounts()))
	require.Equal(t, []int{2, 1}, w(child1.GetCounts()))
	require.Equal(t, 0, len(sl))
	// This is a special case - as dependency has explicitly released parent, the it can only acquire the parent
	require.Equal(t, smachine.Impossible, child1.UseDependency(dep15, 0))

	require.Equal(t, smachine.NotPassed, child1.UseDependency(dep15, smachine.SyncIgnoreFlags))
	require.Equal(t, smachine.NotPassed, master.UseDependency(dep15, smachine.SyncIgnoreFlags))
	require.Equal(t, []int{2, 0}, w(master.GetCounts()))
	require.Equal(t, []int{2, 1}, w(child1.GetCounts()))

	/*** adjust parent and child0 ***/

	ok2 := false

	sl, ok2 = master.AdjustLimit(3, true)
	require.True(t, ok2)
	// NB! Nothing can be released as limit of both children are full
	require.Equal(t, 0, len(sl))

	sl, ok2 = master.AdjustLimit(2, true)
	require.False(t, ok2)
	require.Equal(t, 0, len(sl))

	sl, ok2 = child0.AdjustLimit(2, true)
	require.True(t, ok2)
	require.Equal(t, 0, len(sl))
	require.Equal(t, []int{2, 1}, w(master.GetCounts()))
	require.Equal(t, []int{2, 1}, w(child0.GetCounts()))

	sl, ok2 = master.AdjustLimit(3, true)
	require.True(t, ok2)
	require.Equal(t, 1, len(sl))
	require.Equal(t, smachine.SlotID(2), sl[0].SlotID())
	require.Equal(t, []int{3, 0}, w(master.GetCounts()))
	require.Equal(t, []int{2, 1}, w(child0.GetCounts()))

	sl, ok2 = child0.AdjustLimit(1, true)
	require.False(t, ok2)
	require.Equal(t, 0, len(sl))

	_, sl = dep02.ReleaseAll()
	require.Equal(t, []int{2, 0}, w(master.GetCounts()))
	require.Equal(t, []int{1, 1}, w(child0.GetCounts()))
	require.Equal(t, 0, len(sl))

	/*** adjust parent and child1 with partially released dependency  ***/

	// This adjustment will release a slot from child, and push the slot through parent's open limit
	sl, ok2 = child1.AdjustLimit(3, true)
	require.True(t, ok2)
	require.Equal(t, 1, len(sl))
	require.Equal(t, smachine.SlotID(6), sl[0].SlotID())
	require.Equal(t, []int{3, 0}, w(master.GetCounts()))
	require.Equal(t, []int{3, 0}, w(child1.GetCounts()))

	// Make sure that partially release dependency can't be activated
	_, sl = dep14.ReleaseAll()
	require.Equal(t, []int{2, 0}, w(master.GetCounts()))
	require.Equal(t, []int{2, 0}, w(child1.GetCounts()))
	require.Equal(t, 0, len(sl))

	_, sl = dep16.ReleaseAll()
	require.Equal(t, []int{1, 0}, w(master.GetCounts()))
	require.Equal(t, []int{1, 0}, w(child1.GetCounts()))
	require.Equal(t, 0, len(sl))

	_, sl = dep03.ReleaseAll()
	require.Equal(t, []int{1, 0}, w(master.GetCounts()))
	require.Equal(t, []int{1, 0}, w(child0.GetCounts()))
	require.Equal(t, []int{1, 0}, w(child1.GetCounts()))
	require.Equal(t, 0, len(sl))

	// master has slot 1 active
	// child0 has slot 1 active
	// child1 has slot 5 partially released

	sl, ok2 = master.AdjustLimit(0, true)
	require.False(t, ok2)
	require.Equal(t, 0, len(sl))

	// reacquire the parent, but master is full
	require.Equal(t, smachine.NotPassed, master.UseDependency(dep15, 0))
	require.Equal(t, []int{1, 1}, w(master.GetCounts()))
	require.Equal(t, []int{1, 0}, w(child1.GetCounts()))

	_, sl = dep01.ReleaseAll()
	require.Equal(t, []int{0, 1}, w(master.GetCounts()))
	require.Equal(t, []int{0, 0}, w(child0.GetCounts()))
	require.Equal(t, []int{1, 0}, w(child1.GetCounts()))
	require.Equal(t, 0, len(sl))

	sl, ok2 = master.AdjustLimit(1, true)
	require.True(t, ok2)
	require.Equal(t, []int{1, 0}, w(master.GetCounts()))
	require.Equal(t, []int{1, 0}, w(child1.GetCounts()))
	require.Equal(t, 1, len(sl))
	require.Equal(t, smachine.SlotID(5), sl[0].SlotID())

	_, sl = dep15.ReleaseAll()
	require.Equal(t, []int{0, 0}, w(master.GetCounts()))
	require.Equal(t, []int{0, 0}, w(child0.GetCounts()))
	require.Equal(t, []int{0, 0}, w(child1.GetCounts()))
	require.Equal(t, 0, len(sl))
}
