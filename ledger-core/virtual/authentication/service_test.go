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
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func Test_IsMessageFromVirtualLegitimate_UnexpectedMessageType(t *testing.T) {
	ctx := inslogger.TestContext(t)
	authService := NewService(ctx, nil)

	pdLeft := pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 10, longbits.Bits256{})

	rg := pulse.NewPulseRange([]pulse.Data{pdLeft})

	_, err := authService.IsMessageFromVirtualLegitimate(ctx, 333, reference.Global{}, rg)
	require.EqualError(t, err, "Unexpected message type")
}

func Test_IsMessageFromVirtualLegitimate_TemporaryIgnoreChecking_APIRequests(t *testing.T) {
	ctx := context.Background()
	selfRef := gen.UniqueGlobalRef()
	sender := statemachine.APICaller

	jetCoordinatorMock := jet.NewAffinityHelperMock(t).
		QueryRoleMock.Return([]reference.Global{selfRef}, nil)
	authService := NewService(ctx, jetCoordinatorMock)

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

			refs := gen.UniqueGlobalRefs(3)
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
				QueryRoleMock.Return([]reference.Global{approver}, nil).
				MeMock.Return(selfRef)

			authService := NewService(ctx, jetCoordinatorMock)

			rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

			mustReject, err := authService.IsMessageFromVirtualLegitimate(ctx, testCase.msg, sender, rg)
			require.NoError(t, err)
			require.False(t, mustReject)
		})

		t.Run("Sender_equals_to_current_node:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()

			refs := gen.UniqueGlobalRefs(2)
			sender := refs[0]
			selfRef := refs[1]

			jetCoordinatorMock := jet.NewAffinityHelperMock(t).
				QueryRoleMock.Return([]reference.Global{selfRef}, nil).
				MeMock.Return(sender)

			authService := NewService(ctx, jetCoordinatorMock)

			rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

			token := payload.CallDelegationToken{
				TokenTypeAndFlags: payload.DelegationTokenTypeCall,
				Approver:          selfRef,
			}

			reflect.ValueOf(testCase.msg).MethodByName("Reset").Call([]reflect.Value{})
			insertToken(token, testCase.msg)

			_, err := authService.IsMessageFromVirtualLegitimate(ctx, testCase.msg, sender, rg)
			require.Contains(t, err.Error(), "current node cannot be equal to sender of message with token")
		})

		t.Run("ExpectedVE_not_equals_to_Approver:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()
			selfRef := gen.UniqueGlobalRef()

			refs := gen.UniqueGlobalRefs(2)
			expectedVE := refs[0]
			approver := refs[1]

			jetCoordinatorMock := jet.NewAffinityHelperMock(t).
				QueryRoleMock.Return([]reference.Global{expectedVE}, nil).
				MeMock.Return(selfRef)

			authService := NewService(ctx, jetCoordinatorMock)

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
			refs := gen.UniqueGlobalRefs(2)
			selfRef := refs[0]
			sender := refs[0]

			jetCoordinatorMock := jet.NewAffinityHelperMock(t).
				QueryRoleMock.Return([]reference.Global{sender}, nil).
				MeMock.Return(selfRef)

			authService := NewService(ctx, jetCoordinatorMock)

			mustReject, err := authService.IsMessageFromVirtualLegitimate(ctx, testCase.msg, sender, pulseRange)
			require.NoError(t, err)
			require.False(t, mustReject)

		})

		t.Run("BadSender:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()
			refs := gen.UniqueGlobalRefs(3)
			selfRef := refs[0]
			sender := refs[1]
			badSender := refs[2]

			jetCoordinatorMock := jet.NewAffinityHelperMock(t).
				QueryRoleMock.Return([]reference.Global{sender}, nil).
				MeMock.Return(selfRef)

			authService := NewService(ctx, jetCoordinatorMock)

			_, err := authService.IsMessageFromVirtualLegitimate(ctx, testCase.msg, badSender, pulseRange)
			require.Contains(t, err.Error(), "unexpected sender")
		})

		t.Run("MustReject_if_message_requires_prev_pulse_for_check:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()
			refs := gen.UniqueGlobalRefs(3)
			selfRef := refs[0]
			sender := refs[1]

			jetCoordinatorMock := jet.NewAffinityHelperMock(t).
				QueryRoleMock.Return([]reference.Global{sender}, nil).
				MeMock.Return(selfRef)
			authService := NewService(ctx, jetCoordinatorMock)

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
			refs := gen.UniqueGlobalRefs(2)
			selfRef := refs[0]
			sender := refs[0]

			jetCoordinatorMock := jet.NewAffinityHelperMock(t).
				QueryRoleMock.Return([]reference.Global{sender, sender}, nil).
				MeMock.Return(selfRef)

			authService := NewService(ctx, jetCoordinatorMock)

			rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

			require.Panics(t, func() {
				authService.IsMessageFromVirtualLegitimate(ctx, testCase.msg, sender, rg)
			})
		})

		t.Run("Cannot_calculate_role:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()
			refs := gen.UniqueGlobalRefs(2)
			selfRef := refs[0]
			sender := refs[0]

			calcErrorMsg := "bad calculator"

			jetCoordinatorMock := jet.NewAffinityHelperMock(t).
				QueryRoleMock.Return([]reference.Global{}, throw.New(calcErrorMsg)).
				MeMock.Return(selfRef)

			authService := NewService(ctx, jetCoordinatorMock)

			rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

			_, err := authService.IsMessageFromVirtualLegitimate(ctx, testCase.msg, sender, rg)
			require.Contains(t, err.Error(), "can't calculate role")
			require.Contains(t, err.Error(), calcErrorMsg)
		})
	}
}

func TestService_HasToSendToken(t *testing.T) {
	var (
		selfRef  = gen.UniqueGlobalRef()
		otherRef = gen.UniqueGlobalRef()
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
