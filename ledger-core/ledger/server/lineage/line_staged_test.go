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

	refReason := gen.UniqueGlobalRefWithPulse(base.GetPulseNumber())
	resolver.FindOtherDependencyMock.Expect(refReason).Return(ResolvedDependency{RecordType: tROutboundRequest}, nil)

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

	line.AddBundle(br, &stubTracker{})
}

type stubTracker struct {
	committed bool
}

func (p *stubTracker) IsCommitted() bool {
	return p.committed
}

