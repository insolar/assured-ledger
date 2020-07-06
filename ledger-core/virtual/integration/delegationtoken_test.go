// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func TestDelegationToken_CheckTokenField(t *testing.T) {
	tests := []struct {
		name         string
		testRailID   string
		fakeCaller   bool
		fakeCallee   bool
		fakeOutgoing bool
	}{
		{
			name:         "Fail with wrong caller in token",
			testRailID:   "C5197",
			fakeCaller:   true,
			fakeCallee:   false,
			fakeOutgoing: false,
		},
		{
			name:         "Fail with wrong callee in token",
			testRailID:   "C5198",
			fakeCaller:   false,
			fakeCallee:   true,
			fakeOutgoing: false,
		},
		{
			name:         "Fail with wrong outgoing in token",
			testRailID:   "C5199",
			fakeCaller:   false,
			fakeCallee:   false,
			fakeOutgoing: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Log(test.testRailID)
			t.Skip("https://insolar.atlassian.net/browse/PLAT-588")

			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			jetCoordinatorMock := jet.NewAffinityHelperMock(t)
			auth := authentication.NewService(ctx, jetCoordinatorMock)
			server.ReplaceAuthenticationService(auth)

			var errorFound bool
			logHandler := func(arg interface{}) {
				if err, ok := arg.(error); ok {
					if severity, sok := throw.GetSeverity(err); sok && severity == throw.RemoteBreachSeverity {
						errorFound = true
					}
				}
			}
			logger := utils.InterceptLog(inslogger.FromContext(ctx), logHandler)
			server.OverrideConveyorFactoryLogContext(inslogger.SetLogger(ctx, logger))

			server.Init(ctx)

			jetCaller := server.RandomGlobalWithPulse()
			jetCoordinatorMock.
				MeMock.Return(jetCaller).
				QueryRoleMock.Return([]reference.Global{jetCaller}, nil)

			var (
				isolation = contract.ConstructorIsolation()
				callFlags = payload.BuildCallFlags(isolation.Interference, isolation.State)

				class    = gen.UniqueGlobalRef()
				outgoing = server.BuildRandomOutgoingWithPulse()

				constructorPulse = server.GetPulse().PulseNumber

				delegationToken payload.CallDelegationToken
			)

			delegationToken = server.DelegationToken(reference.NewRecordOf(class, outgoing.GetLocal()), server.GlobalCaller(), outgoing)
			switch {
			case test.fakeCaller:
				delegationToken.Caller = server.RandomGlobalWithPulse()
			case test.fakeCallee:
				delegationToken.Callee = server.RandomGlobalWithPulse()
			case test.fakeOutgoing:
				delegationToken.Outgoing = server.RandomGlobalWithPulse()
			}

			pl := payload.VCallRequest{
				CallType:       payload.CTConstructor,
				CallFlags:      callFlags,
				CallAsOf:       constructorPulse,
				Callee:         class,
				CallSiteMethod: "New",
				CallOutgoing:   outgoing,
				DelegationSpec: delegationToken,
			}
			server.SendPayload(ctx, &pl)
			server.WaitIdleConveyor()

			assert.True(t, errorFound)

			mc.Finish()
		})
	}
}

func insertToken(token payload.CallDelegationToken, msg interface{}) {
	field := reflect.New(reflect.TypeOf(token))
	field.Elem().Set(reflect.ValueOf(token))
	reflect.ValueOf(msg).Elem().FieldByName("DelegationSpec").Set(field.Elem())
}

func TestDelegationToken_CheckFailIfWrongApprover(t *testing.T) {
	t.Log("C5192")
	cases := []struct {
		name string
		msg  interface{}
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
			name: "VStateReport",
			msg:  &payload.VStateReport{},
		},
		{
			name: "VStateRequest",
			msg:  &payload.VStateRequest{},
		},
		{
			name: "VDelegatedCallRequest",
			msg:  &payload.VDelegatedCallRequest{},
		},
		{
			name: "VDelegatedRequestFinished",
			msg:  &payload.VDelegatedRequestFinished{},
		},
	}

	mc := minimock.NewController(t)

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server, ctx := utils.NewUninitializedServer(nil, t)

			jetCoordinatorMock := jet.NewAffinityHelperMock(t)
			auth := authentication.NewService(ctx, jetCoordinatorMock)
			server.ReplaceAuthenticationService(auth)

			var errorFound bool
			logHandler := func(arg interface{}) {
				if err, ok := arg.(error); ok {
					errMsg := err.Error()
					if strings.Contains(errMsg, "token Approver and expectedVE are different") &&
						strings.Contains(errMsg, "illegitimate msg") {
						errorFound = true
					}
				}
			}
			logger := utils.InterceptLog(inslogger.FromContext(ctx), logHandler)
			server.OverrideConveyorFactoryLogContext(inslogger.SetLogger(ctx, logger))

			server.Init(ctx)
			// increment pulse for VStateReport and VDelegatedCallRequest
			server.IncrementPulse(ctx)

			approver := server.RandomGlobalWithPulse()
			fakeApprover := server.RandomGlobalWithPulse()
			jetCoordinatorMock.
				MeMock.Return(fakeApprover).
				QueryRoleMock.Return([]reference.Global{approver}, nil)

			var (
				class    = gen.UniqueGlobalRef()
				outgoing = server.BuildRandomOutgoingWithPulse()
			)

			delegationToken := server.DelegationToken(reference.NewRecordOf(class, outgoing.GetLocal()), server.GlobalCaller(), outgoing)
			reflect.ValueOf(testCase.msg).MethodByName("Reset").Call([]reflect.Value{})
			insertToken(delegationToken, testCase.msg)

			server.SendPayload(ctx, testCase.msg.(payload.Marshaler))
			server.WaitIdleConveyor()

			assert.True(t, errorFound)
			server.Stop()
		})
	}
	mc.Finish()
}

func TestDelegationToken_CheckFailIfSenderEqApprover(t *testing.T) {
	t.Log("C5193")
	cases := []struct {
		name string
		msg  interface{}
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
			name: "VStateReport",
			msg:  &payload.VStateReport{},
		},
		{
			name: "VStateRequest",
			msg:  &payload.VStateRequest{},
		},
		{
			name: "VDelegatedCallRequest",
			msg:  &payload.VDelegatedCallRequest{},
		},
		{
			name: "VDelegatedRequestFinished",
			msg:  &payload.VDelegatedRequestFinished{},
		},
	}

	mc := minimock.NewController(t)

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server, ctx := utils.NewUninitializedServer(nil, t)

			jetCoordinatorMock := jet.NewAffinityHelperMock(t)
			auth := authentication.NewService(ctx, jetCoordinatorMock)
			server.ReplaceAuthenticationService(auth)

			var errorFound bool
			logHandler := func(arg interface{}) {
				if err, ok := arg.(error); ok {
					errMsg := err.Error()
					if severity, sok := throw.GetSeverity(err); sok && severity == throw.FraudSeverity &&
						strings.Contains(errMsg, "sender cannot be approver of the token") &&
						strings.Contains(errMsg, "illegitimate msg") {
						errorFound = true
					}
				}
			}
			logger := utils.InterceptLog(inslogger.FromContext(ctx), logHandler)
			server.OverrideConveyorFactoryLogContext(inslogger.SetLogger(ctx, logger))

			server.Init(ctx)
			// increment pulse for VStateReport and VDelegatedCallRequest
			server.IncrementPulse(ctx)

			jetCoordinatorMock.
				MeMock.Return(server.GlobalCaller()).
				QueryRoleMock.Return([]reference.Global{server.GlobalCaller()}, nil)

			var (
				class    = gen.UniqueGlobalRef()
				outgoing = server.BuildRandomOutgoingWithPulse()
			)

			delegationToken := server.DelegationToken(reference.NewRecordOf(class, outgoing.GetLocal()), server.GlobalCaller(), outgoing)
			reflect.ValueOf(testCase.msg).MethodByName("Reset").Call([]reflect.Value{})
			insertToken(delegationToken, testCase.msg)

			server.SendPayload(ctx, testCase.msg.(payload.Marshaler))
			server.WaitIdleConveyor()

			assert.True(t, errorFound)
			server.Stop()
		})
	}
	mc.Finish()
}

func TestDelegationToken_CheckFailIfZeroDTAndSenderNEqExpectedVE(t *testing.T) {
	t.Log("C5196")
	cases := []struct {
		name string
		msg  interface{}
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
			name: "VStateReport",
			msg:  &payload.VStateReport{},
		},
		{
			name: "VStateRequest",
			msg:  &payload.VStateRequest{},
		},
		{
			name: "VDelegatedCallRequest",
			msg:  &payload.VDelegatedCallRequest{},
		},
		{
			name: "VDelegatedRequestFinished",
			msg:  &payload.VDelegatedRequestFinished{},
		},
	}

	mc := minimock.NewController(t)

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server, ctx := utils.NewUninitializedServer(nil, t)

			jetCoordinatorMock := jet.NewAffinityHelperMock(t)
			auth := authentication.NewService(ctx, jetCoordinatorMock)
			server.ReplaceAuthenticationService(auth)

			var errorFound bool
			logHandler := func(arg interface{}) {
				if err, ok := arg.(error); ok {
					errMsg := err.Error()
					if strings.Contains(errMsg, "unexpected sender") &&
						strings.Contains(errMsg, "illegitimate msg") {
						errorFound = true
					}
				}
			}
			logger := utils.InterceptLog(inslogger.FromContext(ctx), logHandler)
			server.OverrideConveyorFactoryLogContext(inslogger.SetLogger(ctx, logger))

			server.Init(ctx)
			// increment pulse for VStateReport and VDelegatedCallRequest
			server.IncrementPulse(ctx)

			approver := server.RandomGlobalWithPulse()
			jetCoordinatorMock.
				QueryRoleMock.Return([]reference.Global{approver}, nil)

			delegationToken := payload.CallDelegationToken{
				TokenTypeAndFlags: payload.DelegationTokenTypeUninitialized,
			}
			reflect.ValueOf(testCase.msg).MethodByName("Reset").Call([]reflect.Value{})
			insertToken(delegationToken, testCase.msg)

			server.SendPayload(ctx, testCase.msg.(payload.Marshaler))
			server.WaitIdleConveyor()

			assert.True(t, errorFound)
			server.Stop()
		})
	}
	mc.Finish()
}
