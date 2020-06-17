// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package authentication

import (
	"context"
	"reflect"
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

func insertToken(token payload.CallDelegationToken, msg interface{}) {
	field := reflect.New(reflect.TypeOf(token))
	field.Elem().Set(reflect.ValueOf(token))
	reflect.ValueOf(msg).Elem().FieldByName("DelegationSpec").Set(field.Elem())
}

func Test_IsMessageFromVirtualLegitimate_WithToken(t *testing.T) {
	cases := []struct {
		name         string
		msg          interface{}
		testRailCase string
	}{
		{
			name: "VCallRequest",
			msg:  &payload.VCallRequest{},
		},
		{
			name: "VCallResult",
			msg:  &payload.VCallResult{},
		},
		{
			name: "VStateRequest",
			msg:  &payload.VStateRequest{},
		},
		{
			name: "VStateReport",
			msg:  &payload.VStateReport{},
		},
		{
			name: "VDelegatedRequestFinished",
			msg:  &payload.VDelegatedRequestFinished{},
		},
		{
			name: "VDelegatedCallRequest",
			msg:  &payload.VDelegatedCallRequest{},
		},
	}

	for _, testCase := range cases {
		t.Run("HappyPath:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()

			refs := gen.UniqueReferences(3)
			sender := refs[0]
			selfRef := refs[1]
			approver := refs[2]

			token := payload.CallDelegationToken{
				TokenTypeAndFlags: payload.DelegationTokenTypeCall,
				Approver:          approver,
			}

			reflect.ValueOf(testCase.msg).MethodByName("Reset").Call([]reflect.Value{})
			insertToken(token, testCase.msg)

			jetCoordinatorMock := jet.NewAffinityHelperMock(t).
				QueryRoleMock.Return([]reference.Global{approver}, nil)

			authService := NewService(ctx, selfRef, jetCoordinatorMock)

			rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

			mustReject, err := authService.IsMessageFromVirtualLegitimate(ctx, testCase.msg, sender, rg)
			require.NoError(t, err)
			require.False(t, mustReject)
		})

		t.Run("SelfNode Equals to Approver:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()

			refs := gen.UniqueReferences(2)
			sender := refs[0]
			selfRef := refs[1]

			jetCoordinatorMock := jet.NewAffinityHelperMock(t).
				QueryRoleMock.Return([]reference.Global{selfRef}, nil)

			authService := NewService(ctx, selfRef, jetCoordinatorMock)

			rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

			token := payload.CallDelegationToken{
				TokenTypeAndFlags: payload.DelegationTokenTypeCall,
				Approver:          selfRef,
			}

			reflect.ValueOf(testCase.msg).MethodByName("Reset").Call([]reflect.Value{})
			insertToken(token, testCase.msg)

			_, err := authService.IsMessageFromVirtualLegitimate(ctx, testCase.msg, sender, rg)
			require.Contains(t, err.Error(), "selfNode cannot be equal to token Approver")
		})

		t.Run("ExpectedVE not equals to Approver:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()
			selfRef := gen.UniqueReference()

			refs := gen.UniqueReferences(2)
			expectedVE := refs[0]
			approver := refs[1]

			jetCoordinatorMock := jet.NewAffinityHelperMock(t).
				QueryRoleMock.Return([]reference.Global{expectedVE}, nil)

			authService := NewService(ctx, selfRef, jetCoordinatorMock)

			rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

			reflect.ValueOf(testCase.msg).MethodByName("Reset").Call([]reflect.Value{})
			token := payload.CallDelegationToken{
				TokenTypeAndFlags: payload.DelegationTokenTypeCall,
				Approver:          approver,
			}
			insertToken(token, testCase.msg)

			_, err := authService.IsMessageFromVirtualLegitimate(ctx, testCase.msg, expectedVE, rg)
			require.Contains(t, err.Error(), "token Approver and expectedVE are different")
		})
	}

}

func Test_IsMessageFromVirtualLegitimate_WithoutToken(t *testing.T) {
	cases := []struct {
		name             string
		msg              interface{}
		testRailCase     string
		usePreviousPulse bool
	}{
		{
			name: "VCallRequest",
			msg:  &payload.VCallRequest{},
		},
		{
			name: "VCallResult",
			msg:  &payload.VCallResult{},
		},
		{
			name: "VStateRequest",
			msg:  &payload.VStateRequest{},
		},
		{
			name:             "VStateReport",
			msg:              &payload.VStateReport{},
			usePreviousPulse: true,
		},
		{
			name: "VDelegatedRequestFinished",
			msg:  &payload.VDelegatedRequestFinished{},
		},
		{
			name:             "VDelegatedCallRequest",
			msg:              &payload.VDelegatedCallRequest{},
			usePreviousPulse: true,
		},
		{
			name: "VDelegatedCallResponse",
			msg:  &payload.VDelegatedCallResponse{},
		},
	}

	pulseRange := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	for _, testCase := range cases {
		t.Run("HappyPath:"+testCase.name, func(t *testing.T) {
			if testCase.testRailCase != "" {
				t.Log(testCase.testRailCase)
			}

			ctx := context.Background()
			refs := gen.UniqueReferences(2)
			selfRef := refs[0]
			sender := refs[0]

			jetCoordinatorMock := jet.NewAffinityHelperMock(t).
				QueryRoleMock.Return([]reference.Global{sender}, nil)

			authService := NewService(ctx, selfRef, jetCoordinatorMock)

			mustReject, err := authService.IsMessageFromVirtualLegitimate(ctx, testCase.msg, sender, pulseRange)
			require.NoError(t, err)
			require.False(t, mustReject)

		})

		t.Run("BadSender:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()
			refs := gen.UniqueReferences(3)
			selfRef := refs[0]
			sender := refs[1]
			badSender := refs[2]

			jetCoordinatorMock := jet.NewAffinityHelperMock(t).
				QueryRoleMock.Return([]reference.Global{sender}, nil)

			authService := NewService(ctx, selfRef, jetCoordinatorMock)

			_, err := authService.IsMessageFromVirtualLegitimate(ctx, testCase.msg, badSender, pulseRange)
			require.Contains(t, err.Error(), "unexpected sender")
		})

		t.Run("MustReject if message requires prev pulse for check:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()
			refs := gen.UniqueReferences(3)
			selfRef := refs[0]
			sender := refs[1]

			jetCoordinatorMock := jet.NewAffinityHelperMock(t).
				QueryRoleMock.Return([]reference.Global{sender}, nil)
			authService := NewService(ctx, selfRef, jetCoordinatorMock)

			// reject all messages since pulseRange has bad previous delta
			pr := pulse.NewOnePulseRange(pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 0, longbits.Bits256{}))

			mustReject, err := authService.IsMessageFromVirtualLegitimate(ctx, testCase.msg, sender, pr)
			require.NoError(t, err)
			if testCase.usePreviousPulse {
				require.True(t, mustReject)
			} else {
				require.False(t, mustReject)
			}
		})

		t.Run("More then one possible VE:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()
			refs := gen.UniqueReferences(2)
			selfRef := refs[0]
			sender := refs[0]

			jetCoordinatorMock := jet.NewAffinityHelperMock(t).
				QueryRoleMock.Return([]reference.Global{sender, sender}, nil)

			authService := NewService(ctx, selfRef, jetCoordinatorMock)

			rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

			require.Panics(t, func() {
				authService.IsMessageFromVirtualLegitimate(ctx, testCase.msg, sender, rg)
			})
		})

		t.Run("Cannot calculate role:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()
			refs := gen.UniqueReferences(2)
			selfRef := refs[0]
			sender := refs[0]

			calcErrorMsg := "bad calculator"

			jetCoordinatorMock := jet.NewAffinityHelperMock(t).
				QueryRoleMock.Return([]reference.Global{}, throw.New(calcErrorMsg))

			authService := NewService(ctx, selfRef, jetCoordinatorMock)

			rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

			_, err := authService.IsMessageFromVirtualLegitimate(ctx, testCase.msg, sender, rg)
			require.Contains(t, err.Error(), "can't calculate role")
			require.Contains(t, err.Error(), calcErrorMsg)
		})
	}
}
