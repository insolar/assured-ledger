// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package authentication

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var emptyEntropyFn = func() longbits.Bits256 {
	return longbits.Bits256{}
}

// with token

type noTokenHolderMessageMock struct {
	caller reference.Global
}

func (n noTokenHolderMessageMock) GetCaller() reference.Global {
	return n.caller
}

func Test_IsMessageFromVirtualLegitimate_CantGetToken(t *testing.T) {
	ctx := context.Background()
	selfRef := gen.UniqueReference()

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		QueryRoleMock.Return([]reference.Global{selfRef}, nil)

	authService := NewService(ctx, selfRef, jetCoordinatorMock)
	pdLeft := pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 10, longbits.Bits256{})
	rg := pulse.NewPulseRange([]pulse.Data{pdLeft})
	msg := &noTokenHolderMessageMock{caller: selfRef}
	_, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, selfRef, rg)
	require.EqualError(t, err, "message must implement tokenHolder interface")
}

func Test_IsMessageFromVirtualLegitimate_ExpectedVE_NotEqual_Approver(t *testing.T) {
	ctx := context.Background()
	selfRef := gen.UniqueReference()

	refs := gen.UniqueReferences(2)
	expectedVE := refs[0]
	approver := refs[1]

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		QueryRoleMock.Return([]reference.Global{expectedVE}, nil)

	authService := NewService(ctx, selfRef, jetCoordinatorMock)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &payload.VStateRequest{
		DelegationSpec: payload.CallDelegationToken{
			TokenTypeAndFlags: payload.DelegationTokenTypeCall,
			Approver:          approver,
		},
	}

	_, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, expectedVE, rg)
	require.Contains(t, err.Error(), "token Approver and expectedVE are different")
}

func Test_IsMessageFromVirtualLegitimate_SelfNode_Equals_Approver(t *testing.T) {
	ctx := context.Background()

	refs := gen.UniqueReferences(2)
	sender := refs[0]
	selfRef := refs[1]

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		QueryRoleMock.Return([]reference.Global{selfRef}, nil)

	authService := NewService(ctx, selfRef, jetCoordinatorMock)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &payload.VStateRequest{
		DelegationSpec: payload.CallDelegationToken{
			TokenTypeAndFlags: payload.DelegationTokenTypeCall,
			Approver:          selfRef,
		},
	}

	_, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, sender, rg)
	require.Contains(t, err.Error(), "selfNode cannot be equal to token Approver")
}

func Test_IsMessageFromVirtualLegitimate_WithTokenHappyPath(t *testing.T) {
	ctx := context.Background()

	refs := gen.UniqueReferences(3)
	sender := refs[0]
	selfRef := refs[1]
	approver := refs[2]

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		QueryRoleMock.Return([]reference.Global{approver}, nil)

	authService := NewService(ctx, selfRef, jetCoordinatorMock)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &payload.VStateRequest{
		DelegationSpec: payload.CallDelegationToken{
			TokenTypeAndFlags: payload.DelegationTokenTypeCall,
			Approver:          approver,
		},
	}

	mustReject, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, sender, rg)
	require.NoError(t, err)
	require.False(t, mustReject)
}

// without token

func Test_IsMessageFromVirtualLegitimate_UnexpectedMessageType(t *testing.T) {
	ctx := context.Background()
	selfRef := gen.UniqueReference()

	authService := NewService(ctx, selfRef, nil)

	pdLeft := pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 10, longbits.Bits256{})

	rg := pulse.NewPulseRange([]pulse.Data{pdLeft})

	require.PanicsWithValue(t, "Unexpected message type", func() {
		authService.IsMessageFromVirtualLegitimate(ctx, 333, reference.Global{}, rg)
	})
}

func Test_IsMessageFromVirtualLegitimate_MessageWithCurrentExpectedPulse_HappyPath(t *testing.T) {
	ctx := context.Background()
	refs := gen.UniqueReferences(2)
	selfRef := refs[0]
	sender := refs[0]

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		QueryRoleMock.Return([]reference.Global{sender}, nil)

	authService := NewService(ctx, selfRef, jetCoordinatorMock)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &payload.VStateRequest{}

	mustReject, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, sender, rg)
	require.NoError(t, err)
	require.False(t, mustReject)
}

func Test_IsMessageFromVirtualLegitimate_MessageWithCurrentExpectedPulse_BadSender(t *testing.T) {
	ctx := context.Background()
	refs := gen.UniqueReferences(2)
	selfRef := refs[0]
	sender := refs[0]

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		QueryRoleMock.Return([]reference.Global{sender}, nil)

	authService := NewService(ctx, selfRef, jetCoordinatorMock)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &payload.VStateRequest{}

	_, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, reference.Global{}, rg)
	require.Contains(t, err.Error(), "unexpected sender")
}

func Test_IsMessageFromVirtualLegitimate_MessageWithPrevExpectedPulse_HappyPath(t *testing.T) {
	ctx := context.Background()
	refs := gen.UniqueReferences(2)
	selfRef := refs[0]
	sender := refs[0]

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		QueryRoleMock.Return([]reference.Global{sender}, nil)

	authService := NewService(ctx, selfRef, jetCoordinatorMock)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &payload.VStateReport{}

	mustReject, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, sender, rg)
	require.NoError(t, err)
	require.False(t, mustReject)
}

func Test_IsMessageFromVirtualLegitimate_MessageWithPrevExpectedPulse_BadSender(t *testing.T) {
	ctx := context.Background()
	refs := gen.UniqueReferences(2)
	selfRef := refs[0]
	sender := refs[0]

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		QueryRoleMock.Return([]reference.Global{sender}, nil)

	authService := NewService(ctx, selfRef, jetCoordinatorMock)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &payload.VStateReport{}

	_, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, reference.Global{}, rg)
	require.Contains(t, err.Error(), "unexpected sender")
}

func Test_IsMessageFromVirtualLegitimate_MessageWithPrevExpectedPulse_MustReject(t *testing.T) {
	ctx := context.Background()
	selfRef := gen.UniqueReference()
	authService := NewService(ctx, selfRef, nil)

	pr := pulse.NewOnePulseRange(pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 0, longbits.Bits256{}))

	msg := &payload.VStateReport{}

	mustReject, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, reference.Global{}, pr)
	require.NoError(t, err)
	require.True(t, mustReject)
}

func Test_IsMessageFromVirtualLegitimate_CantCalculateRole(t *testing.T) {
	ctx := context.Background()
	refs := gen.UniqueReferences(2)
	selfRef := refs[0]
	sender := refs[0]

	calcErrorMsg := "bad calculator"

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		QueryRoleMock.Return([]reference.Global{}, throw.New(calcErrorMsg))

	authService := NewService(ctx, selfRef, jetCoordinatorMock)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &payload.VStateReport{}

	_, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, sender, rg)
	require.Contains(t, err.Error(), "can't calculate role")
	require.Contains(t, err.Error(), calcErrorMsg)
}

func Test_IsMessageFromVirtualLegitimate_HaveMoreThanOneResponsibleVE(t *testing.T) {
	ctx := context.Background()
	refs := gen.UniqueReferences(2)
	selfRef := refs[0]
	sender := refs[0]

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		QueryRoleMock.Return([]reference.Global{sender, sender}, nil)

	authService := NewService(ctx, selfRef, jetCoordinatorMock)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &payload.VStateReport{}

	require.Panics(t, func() {
		authService.IsMessageFromVirtualLegitimate(ctx, msg, sender, rg)
	})
}

func Test_IsMessageFromVirtualLegitimate_TemporaryIgnoreChecking_APIRequests(t *testing.T) {
	ctx := context.Background()
	selfRef := gen.UniqueReference()
	sender := statemachine.APICaller

	authService := NewService(ctx, selfRef, nil)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &payload.VStateRequest{
		Object: statemachine.APICaller,
	}

	mustReject, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, sender, rg)
	require.NoError(t, err)
	require.False(t, mustReject)
}
