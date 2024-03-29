package authentication

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/application/testwalletapi/statemachine"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func Test_CheckMessageFromAuthorizedVirtual_UnexpectedMessageType(t *testing.T) {
	ctx := instestlogger.TestContext(t)
	authService := NewService(ctx, nil)

	pdLeft := pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 10, longbits.Bits256{})

	rg := pulse.NewPulseRange([]pulse.Data{pdLeft})

	_, err := authService.CheckMessageFromAuthorizedVirtual(ctx, 333, reference.Global{}, rg)
	require.EqualError(t, err, "Unexpected message type")
}

func Test_CheckMessageFromAuthorizedVirtual_TemporaryIgnoreChecking_APIRequests(t *testing.T) {
	ctx := context.Background()
	selfRef := gen.UniqueGlobalRef()
	sender := statemachine.APICaller

	jetCoordinatorMock := affinity.NewHelperMock(t).
		QueryRoleMock.Return([]reference.Global{selfRef}, nil)
	authService := NewService(ctx, jetCoordinatorMock)

	rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	msg := &rms.VStateRequest{
		Object: rms.NewReference(statemachine.APICaller),
	}

	mustReject, err := authService.CheckMessageFromAuthorizedVirtual(ctx, msg, sender, rg)
	require.NoError(t, err)
	require.False(t, mustReject)
}

func insertToken(token rms.CallDelegationToken, msg interface{}) {
	field := reflect.New(reflect.TypeOf(token))
	field.Elem().Set(reflect.ValueOf(token))
	reflect.ValueOf(msg).Elem().FieldByName("DelegationSpec").Set(field.Elem())
}

func Test_CheckMessageFromAuthorizedVirtual_WithToken(t *testing.T) {
	cases := []struct {
		name         string
		msg          interface{}
		testRailCase string
	}{
		{
			name: "VCallRequest",
			msg:  &rms.VCallRequest{},
		},
		{
			name: "VCallResult",
			msg:  &rms.VCallResult{},
		},
		{
			name: "VStateRequest",
			msg:  &rms.VStateRequest{},
		},
		{
			name: "VStateReport",
			msg:  &rms.VStateReport{},
		},
		{
			name: "VDelegatedRequestFinished",
			msg:  &rms.VDelegatedRequestFinished{},
		},
		{
			name: "VDelegatedCallRequest",
			msg:  &rms.VDelegatedCallRequest{},
		},
	}

	for _, testCase := range cases {
		t.Run("HappyPath:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()

			refs := gen.UniqueGlobalRefs(3)
			sender := refs[0]
			selfRef := refs[1]
			approver := refs[2]

			token := rms.CallDelegationToken{
				TokenTypeAndFlags: rms.DelegationTokenTypeCall,
				Approver:          rms.NewReference(approver),
				DelegateTo:        rms.NewReference(sender),
			}

			reflect.ValueOf(testCase.msg).MethodByName("Reset").Call([]reflect.Value{})
			insertToken(token, testCase.msg)

			jetCoordinatorMock := affinity.NewHelperMock(t).
				QueryRoleMock.Return([]reference.Global{approver}, nil).
				MeMock.Return(selfRef)

			authService := NewService(ctx, jetCoordinatorMock)

			rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

			mustReject, err := authService.CheckMessageFromAuthorizedVirtual(ctx, testCase.msg, sender, rg)
			require.NoError(t, err)
			require.False(t, mustReject)
		})

		t.Run("Sender_equals_approver:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()

			sender := gen.UniqueGlobalRef()

			jetCoordinatorMock := affinity.NewHelperMock(t).
				QueryRoleMock.Return([]reference.Global{sender}, nil)

			authService := NewService(ctx, jetCoordinatorMock)

			rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

			token := rms.CallDelegationToken{
				TokenTypeAndFlags: rms.DelegationTokenTypeCall,
				Approver:          rms.NewReference(sender),
				DelegateTo:        rms.NewReference(sender),
			}

			reflect.ValueOf(testCase.msg).MethodByName("Reset").Call([]reflect.Value{})
			insertToken(token, testCase.msg)

			_, err := authService.CheckMessageFromAuthorizedVirtual(ctx, testCase.msg, sender, rg)
			require.Contains(t, err.Error(), "sender cannot be approver of the token")
		})

		t.Run("ExpectedVE_not_equals_to_Approver:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()
			selfRef := gen.UniqueGlobalRef()

			refs := gen.UniqueGlobalRefs(2)
			expectedVE := refs[0]
			approver := refs[1]

			jetCoordinatorMock := affinity.NewHelperMock(t).
				QueryRoleMock.Return([]reference.Global{expectedVE}, nil).
				MeMock.Return(selfRef)

			authService := NewService(ctx, jetCoordinatorMock)

			rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

			reflect.ValueOf(testCase.msg).MethodByName("Reset").Call([]reflect.Value{})
			token := rms.CallDelegationToken{
				TokenTypeAndFlags: rms.DelegationTokenTypeCall,
				Approver:          rms.NewReference(approver),
				DelegateTo:        rms.NewReference(expectedVE),
			}
			insertToken(token, testCase.msg)

			_, err := authService.CheckMessageFromAuthorizedVirtual(ctx, testCase.msg, expectedVE, rg)
			require.Contains(t, err.Error(), "token Approver and expectedVE are different")
		})

		t.Run("Wrong DelegateTo:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()
			selfRef := gen.UniqueGlobalRef()

			refs := gen.UniqueGlobalRefs(2)
			expectedVE := refs[0]
			approver := refs[1]

			jetCoordinatorMock := affinity.NewHelperMock(t).
				QueryRoleMock.Return([]reference.Global{expectedVE}, nil).
				MeMock.Return(selfRef)

			authService := NewService(ctx, jetCoordinatorMock)

			rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

			reflect.ValueOf(testCase.msg).MethodByName("Reset").Call([]reflect.Value{})
			token := rms.CallDelegationToken{
				TokenTypeAndFlags: rms.DelegationTokenTypeCall,
				Approver:          rms.NewReference(approver),
				DelegateTo:        rms.NewReference(gen.UniqueGlobalRef()),
			}
			insertToken(token, testCase.msg)

			_, err := authService.CheckMessageFromAuthorizedVirtual(ctx, testCase.msg, expectedVE, rg)
			require.Contains(t, err.Error(), "token Approver and expectedVE are different")
		})
	}

}

func Test_CheckMessageFromAuthorizedVirtual_WithoutToken(t *testing.T) {
	cases := []struct {
		name         string
		msg          interface{}
		testRailCase string
		mode         rms.AuthSubjectMode
	}{
		{
			name: "VCallRequest",
			msg:  &rms.VCallRequest{},
		},
		{
			name: "VCallResult",
			msg:  &rms.VCallResult{},
		},
		{
			name: "VStateRequest",
			msg:  &rms.VStateRequest{},
		},
		{
			name: "VStateReport",
			msg:  &rms.VStateReport{},
			mode: rms.UsePrevPulse,
		},
		{
			name: "VDelegatedRequestFinished",
			msg:  &rms.VDelegatedRequestFinished{},
		},
		{
			name: "VDelegatedCallRequest",
			msg:  &rms.VDelegatedCallRequest{},
			mode: rms.UsePrevPulse,
		},
		{
			name: "VDelegatedCallResponse",
			msg:  &rms.VDelegatedCallResponse{},
		},
		{
			name: "VFindCallRequest",
			msg:  &rms.VFindCallRequest{},
		},
		{
			name: "VFindCallResponse",
			msg:  &rms.VFindCallResponse{},
			mode: rms.UseAnyPulse,
		},
	}

	pulseRange := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

	for _, testCase := range cases {
		t.Run("HappyPath:"+testCase.name, func(t *testing.T) {
			if testCase.testRailCase != "" {
				insrail.LogCase(t, testCase.testRailCase)
			}

			ctx := context.Background()
			refs := gen.UniqueGlobalRefs(2)
			selfRef := refs[0]
			sender := refs[0]

			jetCoordinatorMock := affinity.NewHelperMock(t).
				QueryRoleMock.Return([]reference.Global{sender}, nil).
				MeMock.Return(selfRef)

			authService := NewService(ctx, jetCoordinatorMock)

			mustReject, err := authService.CheckMessageFromAuthorizedVirtual(ctx, testCase.msg, sender, pulseRange)
			require.NoError(t, err)
			require.False(t, mustReject)

		})

		t.Run("BadSender:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()
			refs := gen.UniqueGlobalRefs(3)
			selfRef := refs[0]
			sender := refs[1]
			badSender := refs[2]

			jetCoordinatorMock := affinity.NewHelperMock(t).
				QueryRoleMock.Return([]reference.Global{sender}, nil).
				MeMock.Return(selfRef)

			authService := NewService(ctx, jetCoordinatorMock)

			mustReject, err := authService.CheckMessageFromAuthorizedVirtual(ctx, testCase.msg, badSender, pulseRange)
			if testCase.mode == rms.UseAnyPulse {
				require.NoError(t, err)
				require.False(t, mustReject)
			} else {
				require.Contains(t, err.Error(), "unexpected sender")
				require.False(t, mustReject)
			}
		})

		t.Run("MustReject_if_message_requires_prev_pulse_for_check:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()
			refs := gen.UniqueGlobalRefs(3)
			selfRef := refs[0]
			sender := refs[1]

			jetCoordinatorMock := affinity.NewHelperMock(t).
				QueryRoleMock.Return([]reference.Global{sender}, nil).
				MeMock.Return(selfRef)
			authService := NewService(ctx, jetCoordinatorMock)

			// reject all messages since pulseRange has bad previous delta
			pr := pulse.NewOnePulseRange(pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 0, longbits.Bits256{}))

			mustReject, err := authService.CheckMessageFromAuthorizedVirtual(ctx, testCase.msg, sender, pr)
			require.NoError(t, err)
			if testCase.mode == rms.UsePrevPulse {
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

			jetCoordinatorMock := affinity.NewHelperMock(t).
				QueryRoleMock.Return([]reference.Global{sender, sender}, nil).
				MeMock.Return(selfRef)

			authService := NewService(ctx, jetCoordinatorMock)

			rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

			if testCase.mode == rms.UseAnyPulse {
				mustReject, err := authService.CheckMessageFromAuthorizedVirtual(ctx, testCase.msg, sender, rg)
				require.NoError(t, err)
				require.False(t, mustReject)
			} else {
				require.Panics(t, func() {
					authService.CheckMessageFromAuthorizedVirtual(ctx, testCase.msg, sender, rg)
				})
			}
		})

		t.Run("Cannot_calculate_role:"+testCase.name, func(t *testing.T) {
			ctx := context.Background()
			refs := gen.UniqueGlobalRefs(2)
			selfRef := refs[0]
			sender := refs[0]

			calcErrorMsg := "bad calculator"

			jetCoordinatorMock := affinity.NewHelperMock(t).
				QueryRoleMock.Return([]reference.Global{}, throw.New(calcErrorMsg)).
				MeMock.Return(selfRef)

			authService := NewService(ctx, jetCoordinatorMock)

			rg := pulse.NewSequenceRange([]pulse.Data{pulse.NewPulsarData(pulse.MinTimePulse<<1, 10, 1, longbits.Bits256{})})

			mustReject, err := authService.CheckMessageFromAuthorizedVirtual(ctx, testCase.msg, sender, rg)
			if testCase.mode == rms.UseAnyPulse {
				require.NoError(t, err)
				require.False(t, mustReject)
			} else {
				require.Contains(t, err.Error(), "can't calculate role")
				require.Contains(t, err.Error(), calcErrorMsg)
				require.False(t, mustReject)
			}
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
				ctx     = instestlogger.TestContext(t)
				affMock = affinity.NewHelperMock(t).MeMock.Return(selfRef)
			)

			authService := NewService(ctx, affMock)

			hasToSendToken := authService.HasToSendToken(rms.CallDelegationToken{
				Approver: rms.NewReference(test.approver),
				Caller:   rms.NewReference(selfRef),
			})

			require.Equal(t, test.rv, hasToSendToken)
		})
	}
}
