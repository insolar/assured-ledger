package lineage

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func r(root reference.Holder, ref, prev reference.LocalHolder, rt RecordType, reason reference.Holder) (rec Record) {
	rec.Excerpt.RecordType = uint32(rt)
	rec.Excerpt.ReasonRef.Set(reason)
	b := root.GetBase()
	rec.Excerpt.RootRef.Set(root)
	rec.regReq = &rms.LRegisterRequest{}
	rec.RecRef = reference.New(b, ref.GetLocal())
	rec.regReq.AnticipatedRef.Set(rec.RecRef)
	rec.Excerpt.PrevRef.Set(reference.New(b, prev.GetLocal()))
	return
}

func rStart(base reference.LocalHolder, reason reference.Holder) (rec Record) {
	rec.Excerpt.RecordType = uint32(tRLifelineStart)
	rec.Excerpt.ReasonRef.Set(reason)
	b := base.GetLocal()
	rec.regReq = &rms.LRegisterRequest{}
	rec.RecRef = reference.NewSelf(b)
	rec.regReq.AnticipatedRef.Set(rec.RecRef)
	return
}

func describe(br *BundleResolver) interface{} {
	return br.errors
}

func TestBundleResolver_Create(t *testing.T) {
	base := gen.UniqueLocalRef()

	resolver := NewLineResolverMock(t)
	resolver.getLineBaseMock.Return(base)
	resolver.getLocalPNMock.Return(base.GetPulseNumber())
	resolver.getNextFilNoMock.Return(1)
	resolver.getNextRecNoMock.Return(1)
	resolver.findCollisionMock.Return(0, nil)
	resolver.findFilamentMock.Return(0, ResolvedDependency{})
	resolver.findLocalDependencyMock.Return(0, 0, ResolvedDependency{})
	resolver.findFilamentMock.Return(0, ResolvedDependency{})

	refReason := gen.UniqueGlobalRefWithPulse(base.GetPulseNumber())
	resolver.findOtherDependencyMock.Expect(refReason).Return(ResolvedDependency{RecordType: tROutboundRequest}, nil)

	br := newBundleResolver(resolver, GetRecordPolicy)


	require.True(t, br.Add(rStart(base, refReason)), describe(br))

	baseRef := reference.NewSelf(base)

	refInitMem := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(baseRef, refInitMem, base, tRLineMemoryInit, nil)), describe(br))

	refActivate := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(baseRef, refActivate, refInitMem, tRLineActivate, nil)), describe(br))

	refInbound1 := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(baseRef, refInbound1, refActivate, tRLineInboundRequest, refReason)), describe(br))

	require.Equal(t, 4, len(br.records))

	for i, r := range br.records {
		require.Equal(t, filamentNo(1), r.filNo)
		rn := recordNo(i)
		require.Equal(t, rn + 1, r.recordNo)
		require.Equal(t, rn, r.prev)
		if i + 1 < len(br.records) {
			require.Equal(t, rn + 2, r.next)
		}
	}
}

func TestBundleResolver_CreateWithCalls(t *testing.T) {
	base := gen.UniqueLocalRef()

	resolver := NewLineResolverMock(t)
	resolver.getLineBaseMock.Return(base)
	resolver.getLocalPNMock.Return(base.GetPulseNumber())
	resolver.getNextFilNoMock.Return(1)
	resolver.getNextRecNoMock.Return(1)
	resolver.findCollisionMock.Return(0, nil)
	resolver.findFilamentMock.Return(0, ResolvedDependency{})
	resolver.findLocalDependencyMock.Return(0, 0, ResolvedDependency{})
	resolver.findFilamentMock.Return(0, ResolvedDependency{})

	refReason := gen.UniqueGlobalRefWithPulse(base.GetPulseNumber())
	resolver.findOtherDependencyMock.Expect(refReason).Return(ResolvedDependency{RecordType: tROutboundRequest}, nil)

	br := newBundleResolver(resolver, GetRecordPolicy)

	require.True(t, br.Add(rStart(base, refReason)), describe(br))

	baseRef := reference.NewSelf(base)

	refInbound1 := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(baseRef, refInbound1, base, tRLineInboundRequest, reference.NewSelf(base))), describe(br))

	fil1Ref := reference.New(base, refInbound1)

	refOutboundRq := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(fil1Ref, refOutboundRq, refInbound1, tROutboundRequest, nil)), describe(br))

	refOutboundRs := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(fil1Ref, refOutboundRs, refOutboundRq, tROutboundResponse, nil)), describe(br))

	refInbound1Rs := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(fil1Ref, refInbound1Rs, refOutboundRs, tRInboundResponse, nil)), describe(br))

	refMem := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())

	rec := r(baseRef, refMem, refInbound1, tRLineMemory, nil)
	rec.Excerpt.RejoinRef.Set(reference.New(base, refInbound1Rs))
	require.True(t, br.Add(rec), describe(br))

	refActivate := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())

	rec = r(baseRef, refActivate, refMem, tRLineActivate, nil)
	rec.Excerpt.RejoinRef.Set(reference.New(base, refInbound1Rs))
	require.True(t, br.Add(rec), describe(br))

	require.Equal(t, 7, len(br.records))

	for i, r := range br.records {
		require.Equal(t, recordNo(i) + 1, r.recordNo)
	}

	require.Equal(t, filamentNo(1), br.records[0].filNo) // LifelineStart
	require.Equal(t, filamentNo(1), br.records[1].filNo) // LineInbound

	require.Equal(t, filamentNo(0), br.records[2].filNo) // OutboundRq
	require.Equal(t, filamentNo(0), br.records[3].filNo) // OutboundRs
	require.Equal(t, filamentNo(0), br.records[4].filNo) // InboundRs

	require.Equal(t, filamentNo(1), br.records[5].filNo) // LineMemory
	require.Equal(t, filamentNo(1), br.records[6].filNo) // LineActivate

	require.Equal(t, recordNo(0), br.records[0].prev)
	require.Equal(t, recordNo(1), br.records[1].prev)
	require.Equal(t, recordNo(2), br.records[2].prev)
	require.Equal(t, recordNo(3), br.records[3].prev)
	require.Equal(t, recordNo(4), br.records[4].prev)
	require.Equal(t, recordNo(2), br.records[5].prev)
	require.Equal(t, recordNo(6), br.records[6].prev)

	require.Equal(t, recordNo(2), br.records[0].next)
	require.Equal(t, recordNo(6), br.records[1].next)
	require.Equal(t, recordNo(4), br.records[2].next)
	require.Equal(t, recordNo(5), br.records[3].next)
	require.Equal(t, deadFilament, br.records[4].next)
	require.Equal(t, recordNo(7), br.records[5].next)
	require.Equal(t, recordNo(0), br.records[6].next)
}


