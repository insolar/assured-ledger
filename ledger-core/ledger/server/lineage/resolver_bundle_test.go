// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

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
	rec.RegRecord = &rms.LRegisterRequest{}
	rec.RegRecord.AnticipatedRef.Set(reference.New(b, ref.GetLocal()))
	rec.Excerpt.PrevRef.Set(reference.New(b, prev.GetLocal()))
	return
}

func rStart(base reference.LocalHolder, reason reference.Holder) (rec Record) {
	rec.Excerpt.RecordType = uint32(tRLifelineStart)
	rec.Excerpt.ReasonRef.Set(reason)
	b := base.GetLocal()
	rec.RegRecord = &rms.LRegisterRequest{}
	rec.RegRecord.AnticipatedRef.Set(reference.NewSelf(b))
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

	br := newBundleResolver(resolver, GetRecordPolicy)

	resolver.findLineAnyDependencyMock.Return(ResolvedDependency{RecordType: tROutboundRequest})

	require.True(t, br.Add(rStart(base, refReason)), describe(br))

	baseRef := reference.NewSelf(base)

	refInitMem := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(baseRef, refInitMem, base, tRLineMemoryInit, nil)), describe(br))

	refActivate := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(baseRef, refActivate, refInitMem, tRLineActivate, nil)), describe(br))

	refInbound1 := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(baseRef, refInbound1, refActivate, tRLineInboundRequest, refReason)), describe(br))
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

	br := newBundleResolver(resolver, GetRecordPolicy)

	resolver.findLineAnyDependencyMock.Return(ResolvedDependency{RecordType: tROutboundRequest})

	require.True(t, br.Add(rStart(base, refReason)), describe(br))

	baseRef := reference.NewSelf(base)

	refInbound1 := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(baseRef, refInbound1, base, tRLineInboundRequest, reference.NewSelf(base))), describe(br))

	fil1Ref := reference.New(base, refInbound1)

	refOutboundRq := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(fil1Ref, refOutboundRq, refInbound1, tROutboundRequest, nil)), describe(br))

	refOutboundRs := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(fil1Ref, refOutboundRs, refOutboundRq, tROutboundResponse, nil)), describe(br))

	refMem := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())

	rec := r(baseRef, refMem, refInbound1, tRLineMemory, nil)
	rec.Excerpt.RejoinRef.Set(reference.New(base, refOutboundRs))
	require.True(t, br.Add(rec), describe(br))

	refActivate := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())

	rec = r(baseRef, refActivate, refMem, tRLineActivate, nil)
	rec.Excerpt.RejoinRef.Set(reference.New(base, refOutboundRs))
	require.True(t, br.Add(rec), describe(br))
}

