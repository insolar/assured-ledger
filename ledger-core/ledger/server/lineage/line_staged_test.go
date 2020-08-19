// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/ledger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestLineStages_Create(t *testing.T) {
	base := gen.UniqueLocalRef()
	resolver := NewDependencyResolverMock(t)

	line := &LineStages{ base: base, pn: base.GetPulseNumber(), cache: resolver }
	br := line.NewBundle()

	refReason := gen.UniqueGlobalRefWithPulse(base.GetPulseNumber())
	resolver.FindOtherDependencyMock.Expect(refReason).Return(ResolvedDependency{RecordType: tROutboundRequest}, nil)

	require.True(t, br.Add(rStart(base, refReason)), describe(br))

	baseRef := reference.NewSelf(base)

	refInitMem := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(baseRef, refInitMem, base, tRLineMemoryInit, nil)), describe(br))

	refActivate := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(baseRef, refActivate, refInitMem, tRLineActivate, nil)), describe(br))

	refInbound1 := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(baseRef, refInbound1, refActivate, tRLineInboundRequest, refReason)), describe(br))

	require.True(t, line.addBundle(br, &stubTracker{}), describe(br))

	verifySequences(t, line)
}

func TestLineStages_CreateWithCalls(t *testing.T) {
	base := gen.UniqueLocalRef()
	resolver := NewDependencyResolverMock(t)

	line := &LineStages{ base: base, pn: base.GetPulseNumber(), cache: resolver }
	br := line.NewBundle()

	reasons := map[reference.Global]struct{}{}

	resolver.FindOtherDependencyMock.Set(func(ref reference.Holder) (ResolvedDependency, error) {
		if _, ok := reasons[reference.Copy(ref)]; ok {
			return ResolvedDependency{RecordType: tROutboundRequest}, nil
		}
		return ResolvedDependency{}, nil
	})

	refReason1 := gen.UniqueGlobalRefWithPulse(base.GetPulseNumber())
	reasons[refReason1] = struct {}{}

	require.True(t, br.Add(rStart(base, refReason1)), describe(br))

	last := fillBundleWithOrderedCall(t, base, base, br, reference.NewSelf(base), true)

	st1 := &stubTracker{}
	require.True(t, line.addBundle(br, st1), describe(br))

	br = line.NewBundle()

	refReason2 := gen.UniqueGlobalRefWithPulse(base.GetPulseNumber())
	refReason3 := gen.UniqueGlobalRefWithPulse(base.GetPulseNumber())
	reasons[refReason2] = struct {}{}
	reasons[refReason3] = struct {}{}

	fillBundleWithOrderedCall(t, base, last, br, refReason2, false)

	br2 := line.NewBundle()
	fillBundleWithUnorderedCall(t, base, last, br2, refReason3)
	st2 := &stubTracker{}
	require.True(t, line.addBundle(br2, st2), describe(br2))

	// not conflicting bundles can be added in any order
	st3 := &stubTracker{}
	require.True(t, line.addBundle(br, st3), describe(br))

	require.Equal(t, recordNo(17), line.getNextRecNo())
	require.Equal(t, stageNo(1), line.earliest.seqNo)
	require.Equal(t, stageNo(3), line.latest.seqNo)

	line.TrimCommittedStages()

	require.Equal(t, stageNo(1), line.earliest.seqNo)
	require.NotNil(t, line.earliest.tracker)

	st1.ready = 7
	line.TrimCommittedStages()

	require.Equal(t, stageNo(2), line.earliest.seqNo)
	require.NotNil(t, line.earliest.tracker)

	st2.ready = 4
	st3.ready = 5
	line.TrimCommittedStages()

	require.Equal(t, stageNo(3), line.earliest.seqNo)
	require.Nil(t, line.earliest.tracker)

	verifySequences(t, line)
}

func TestLineStages_Rollback(t *testing.T) {
	base := gen.UniqueLocalRef()
	resolver := NewDependencyResolverMock(t)

	line := &LineStages{ base: base, pn: base.GetPulseNumber(), cache: resolver }
	br := line.NewBundle()

	reasons := map[reference.Global]struct{}{}

	resolver.FindOtherDependencyMock.Set(func(ref reference.Holder) (ResolvedDependency, error) {
		if _, ok := reasons[reference.Copy(ref)]; ok {
			return ResolvedDependency{RecordType: tROutboundRequest}, nil
		}
		return ResolvedDependency{}, nil
	})

	refReason1 := gen.UniqueGlobalRefWithPulse(base.GetPulseNumber())
	reasons[refReason1] = struct {}{}

	require.True(t, br.Add(rStart(base, refReason1)), describe(br))

	last := fillBundleWithOrderedCall(t, base, base, br, reference.NewSelf(base), true)

	st1 := &stubTracker{}
	require.True(t, line.addBundle(br, st1), describe(br))
	verifySequences(t, line)


	refReason2 := gen.UniqueGlobalRefWithPulse(base.GetPulseNumber())
	refReason3 := gen.UniqueGlobalRefWithPulse(base.GetPulseNumber())
	refReason4 := gen.UniqueGlobalRefWithPulse(base.GetPulseNumber())
	reasons[refReason2] = struct {}{}
	reasons[refReason3] = struct {}{}
	reasons[refReason4] = struct {}{}


	br = line.NewBundle()
	fillBundleWithUnorderedCall(t, base, last, br, refReason2)
	st2 := &stubTracker{}
	require.True(t, line.addBundle(br, st2), describe(br))
	verifySequences(t, line)

	trimAt := line.getNextRecNo()

	br = line.NewBundle()
	fillBundleWithOrderedCall(t, base, last, br, refReason3, false)
	st3 := &stubTracker{}
	require.True(t, line.addBundle(br, st3), describe(br))
	verifySequences(t, line)

	require.Equal(t, recordNo(17), line.getNextRecNo())

	st1.ready = 7
	st2.ready = 4
	line.RollbackUncommittedRecords()

	require.Equal(t, trimAt, line.getNextRecNo())
	require.Equal(t, stageNo(2), line.earliest.seqNo)
	require.Equal(t, stageNo(2), line.latest.seqNo)
	require.Nil(t, line.earliest.tracker)

	verifySequences(t, line)

	br = line.NewBundle()
	fillBundleWithUnorderedCall(t, base, last, br, refReason4)
	st4 := &stubTracker{}
	require.True(t, line.addBundle(br, st4), describe(br))
	st4.ready = 4

	line.RollbackUncommittedRecords()
	require.Equal(t, recordNo(16), line.getNextRecNo())

	verifySequences(t, line)
}


func fillBundleWithOrderedCall(t *testing.T, base, prev reference.Local, br *BundleResolver, orderedReasonRef reference.Holder, addActivate bool) reference.Local {
	baseRef := reference.NewSelf(base)

	refInboundOrd := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(baseRef, refInboundOrd, prev, tRLineInboundRequest, orderedReasonRef)), describe(br))

	fil1Ref := reference.New(base, refInboundOrd)

	refOutboundRq := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(fil1Ref, refOutboundRq, refInboundOrd, tROutboundRequest, nil)), describe(br))

	refOutboundRs := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(fil1Ref, refOutboundRs, refOutboundRq, tROutboundResponse, nil)), describe(br))

	refInboundOrdRs := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(fil1Ref, refInboundOrdRs, refOutboundRs, tRInboundResponse, nil)), describe(br))

	refMem := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())

	rec := r(baseRef, refMem, refInboundOrd, tRLineMemory, nil)
	rec.Excerpt.RejoinRef.Set(reference.New(base, refInboundOrdRs))
	require.True(t, br.Add(rec), describe(br))

	if !addActivate {
		return refMem
	}

	refActivate := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())

	rec = r(baseRef, refActivate, refMem, tRLineActivate, nil)
	rec.Excerpt.RejoinRef.Set(reference.New(base, refInboundOrdRs))
	require.True(t, br.Add(rec), describe(br))
	return refActivate
}

func fillBundleWithUnorderedCall(t *testing.T, base, prev reference.Local, br *BundleResolver, unorderedReasonRef reference.Holder) reference.Local {
	baseRef := reference.NewSelf(base)

	refInboundUnord := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	fil2Ref := reference.New(base, refInboundUnord)
	refOutboundRs2 := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())

	require.True(t, br.Add(r(baseRef, refInboundUnord, prev, tRInboundRequest, unorderedReasonRef)), describe(br))

	refOutboundRq2 := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(fil2Ref, refOutboundRq2, refInboundUnord, tROutboundRequest, nil)), describe(br))

	require.True(t, br.Add(r(fil2Ref, refOutboundRs2, refOutboundRq2, tROutboundResponse, nil)), describe(br))

	refInboundUnordRs := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(fil2Ref, refInboundUnordRs, refOutboundRs2, tRInboundResponse, nil)), describe(br))

	return refInboundUnordRs
}

func verifySequences(t *testing.T, line *LineStages) {
	for i := range line.records[0] {
		recNo := recordNo(i + 1)
		rec := line.get(recNo)
		// fmt.Printf("%2d: %+v\n", recNo, *rec)
		require.NotNil(t, rec)

		require.Equal(t, recNo, rec.recordNo, i)

		if i == 0 {
			require.Zero(t, rec.prev, i)
			require.Equal(t, filamentNo(1), rec.filNo, i)
		} else {
			require.NotZero(t, rec.prev, i)
			require.Less(t, uint32(rec.prev), uint32(recNo), i)
		}

		require.NotZero(t, rec.filNo, i)
		if rec.next != 0 {
			require.Greater(t, uint32(rec.next), uint32(recNo), i)
		} else {
			latest := line.latest.filaments[rec.filNo - 1].latest
			require.Equal(t, latest, recNo, i)
		}
	}

	// make sure that trunk is open
	require.Equal(t, recordNo(0), line.get(line.latest.filaments[0].latest).next)
}

type stubTracker struct {
	ready int
}

func (p *stubTracker) GetFutureAllocation() (isReady bool, allocations []ledger.DirectoryIndex) {
	if p.ready == 0 {
		return false, nil
	}
	return true, make([]ledger.DirectoryIndex, p.ready)
}
