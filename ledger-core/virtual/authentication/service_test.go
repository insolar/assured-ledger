// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package authentication

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func Test_IsMessageFromVirtualLegitimate_CantGetToken(t *testing.T) {
	ctx := inslogger.TestContext(t)

	authService := NewService(ctx, nil)
	_, err := authService.IsMessageFromVirtualLegitimate(ctx, 33, reference.Global{}, nil)
	require.EqualError(t, err, "message must implement tokenHolder interface")
}

type tokenHolderMock struct {
	token payload.CallDelegationToken
}

func (h tokenHolderMock) GetDelegationSpec() payload.CallDelegationToken {
	return h.token
}

var emptyEntropyFn = func() longbits.Bits256 {
	return longbits.Bits256{}
}

func Test_IsMessageFromVirtualLegitimate_UnexpectedMessageType(t *testing.T) {
	ctx := inslogger.TestContext(t)
	authService := NewService(ctx, nil)

	pdLeft := pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 10, longbits.Bits256{})

	rg := pulse.NewPulseRange([]pulse.Data{pdLeft})

	require.PanicsWithValue(t, "Unexpected message type", func() {
		authService.IsMessageFromVirtualLegitimate(ctx, tokenHolderMock{}, reference.Global{}, rg)
	})
}

func Test_IsMessageFromVirtualLegitimate_MessageWithCurrentExpectedPulse_HappyPath(t *testing.T) {
	ctx := inslogger.TestContext(t)
	sender := gen.UniqueReference()

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		QueryRoleMock.Return([]reference.Global{sender}, nil)

	authService := NewService(ctx, jetCoordinatorMock)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &payload.VStateRequest{}

	mustReject, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, sender, rg)
	require.NoError(t, err)
	require.False(t, mustReject)
}

func Test_IsMessageFromVirtualLegitimate_MessageWithCurrentExpectedPulse_BadSender(t *testing.T) {
	ctx := inslogger.TestContext(t)
	sender := gen.UniqueReference()

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		QueryRoleMock.Return([]reference.Global{sender}, nil)

	authService := NewService(ctx, jetCoordinatorMock)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &payload.VStateRequest{}

	_, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, reference.Global{}, rg)
	require.Contains(t, err.Error(), "unexpected sender")
}

func Test_IsMessageFromVirtualLegitimate_MessageWithPrevExpectedPulse_HappyPath(t *testing.T) {
	ctx := inslogger.TestContext(t)
	sender := gen.UniqueReference()

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		QueryRoleMock.Return([]reference.Global{sender}, nil)

	authService := NewService(ctx, jetCoordinatorMock)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &payload.VStateReport{}

	mustReject, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, sender, rg)
	require.NoError(t, err)
	require.False(t, mustReject)
}

func Test_IsMessageFromVirtualLegitimate_MessageWithPrevExpectedPulse_BadSender(t *testing.T) {
	ctx := inslogger.TestContext(t)
	sender := gen.UniqueReference()

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		QueryRoleMock.Return([]reference.Global{sender}, nil)

	authService := NewService(ctx, jetCoordinatorMock)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &payload.VStateReport{}

	_, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, reference.Global{}, rg)
	require.Contains(t, err.Error(), "unexpected sender")
}

func Test_IsMessageFromVirtualLegitimate_MessageWithPrevExpectedPulse_MustReject(t *testing.T) {
	ctx := inslogger.TestContext(t)
	authService := NewService(ctx, nil)

	pr := pulse.NewOnePulseRange(pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 0, longbits.Bits256{}))

	msg := &payload.VStateReport{}

	mustReject, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, reference.Global{}, pr)
	require.NoError(t, err)
	require.True(t, mustReject)
}

func Test_IsMessageFromVirtualLegitimate_CantCalculateRole(t *testing.T) {
	ctx := inslogger.TestContext(t)
	sender := gen.UniqueReference()

	calcErrorMsg := "bad calculator"

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		QueryRoleMock.Return([]reference.Global{}, throw.New(calcErrorMsg))

	authService := NewService(ctx, jetCoordinatorMock)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &payload.VStateReport{}

	_, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, sender, rg)
	require.Contains(t, err.Error(), "can't calculate role")
	require.Contains(t, err.Error(), calcErrorMsg)
}

func Test_IsMessageFromVirtualLegitimate_HaveMoreThanOneResponsibleVE(t *testing.T) {
	ctx := inslogger.TestContext(t)
	sender := gen.UniqueReference()

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		QueryRoleMock.Return([]reference.Global{sender, sender}, nil)

	authService := NewService(ctx, jetCoordinatorMock)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &payload.VStateReport{}

	require.Panics(t, func() {
		_, _ = authService.IsMessageFromVirtualLegitimate(ctx, msg, sender, rg)
	})
}

func Test_IsMessageFromVirtualLegitimate_TemporaryIgnoreChecking_APIRequests(t *testing.T) {
	ctx := inslogger.TestContext(t)
	sender := statemachine.APICaller

	authService := NewService(ctx, nil)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &payload.VStateRequest{
		Object: statemachine.APICaller,
	}

	mustReject, err := authService.IsMessageFromVirtualLegitimate(ctx, msg, sender, rg)
	require.NoError(t, err)
	require.False(t, mustReject)
}

func TestService_HasToSendToken(t *testing.T) {
	var (
		selfRef  = gen.UniqueReference()
		otherRef = gen.UniqueReference()
	)
	tests := []struct {
		name     string
		approver reference.Global
		rv       bool
	}{
		{
			name:     "Self_Ignore",
			approver: selfRef,
			rv:       false,
		},
		{
			name:     "Other_Approve",
			approver: otherRef,
			rv:       true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				ctx     = inslogger.TestContext(t)
				affMock = jet.NewAffinityHelperMock(t).MeMock.Return(selfRef)
			)

			authService := NewService(ctx, affMock)

			hasToSendToken := authService.HasToSendToken(payload.CallDelegationToken{
				Approver: test.approver,
				Caller:   selfRef,
			})

			require.Equal(t, test.rv, hasToSendToken)
		})
	}
}
