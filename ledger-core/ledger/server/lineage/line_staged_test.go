// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func TestLineStages_Create(t *testing.T) {
	base := gen.UniqueLocalRef()
	resolver := NewDependencyResolverMock(t)

	line := LineStages{ base: base, pn: base.GetPulseNumber(), cache: resolver }
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

	line.AddBundle(br, &stubTracker{})
}

func TestLineStages_CreateWithCalls(t *testing.T) {
	base := gen.UniqueLocalRef()
	resolver := NewDependencyResolverMock(t)

	line := LineStages{ base: base, pn: base.GetPulseNumber(), cache: resolver }
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

	line.AddBundle(br, &stubTracker{})

	br = line.NewBundle()

	refReason2 := gen.UniqueGlobalRefWithPulse(base.GetPulseNumber())
	refReason3 := gen.UniqueGlobalRefWithPulse(base.GetPulseNumber())
	reasons[refReason2] = struct {}{}
	reasons[refReason3] = struct {}{}

	fillBundleWithOrderedCall(t, base, last, br, refReason2, false)

	br2 := line.NewBundle()
	fillBundleWithUnorderedCall(t, base, last, br2, refReason3)
	line.AddBundle(br2, &stubTracker{})

	// not conflicting bundles can be added in any order
	line.AddBundle(br, &stubTracker{})
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

	require.True(t, br.Add(r(baseRef, refInboundUnord, base, tRInboundRequest, unorderedReasonRef)), describe(br))

	refOutboundRq2 := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(fil2Ref, refOutboundRq2, refInboundUnord, tROutboundRequest, nil)), describe(br))

	require.True(t, br.Add(r(fil2Ref, refOutboundRs2, refOutboundRq2, tROutboundResponse, nil)), describe(br))

	refInboundUnordRs := gen.UniqueLocalRefWithPulse(base.GetPulseNumber())
	require.True(t, br.Add(r(fil2Ref, refInboundUnordRs, refOutboundRs2, tRInboundResponse, nil)), describe(br))

	return refInboundUnordRs
}

type stubTracker struct {
	committed bool
}

func (p *stubTracker) IsCommitted() bool {
	return p.committed
}

