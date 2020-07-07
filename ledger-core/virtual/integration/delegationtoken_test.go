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

type veSetMode int

const (
	veSetNone veSetMode = iota
	veSetServer
	veSetFake
)

type testCase struct {
	name          string
	testRailID    string
	zeroToken     bool
	approverVE    veSetMode
	currentVE     veSetMode
	errorMessages []string
	errorSeverity throw.Severity
}

func TestDelegationToken_IsMessageFromVirtualLegitimate(t *testing.T) {
	cases := []testCase{
		{
			name:       "Fail if DT is zero and sender not eq expectedVE",
			testRailID: "C5196",
			zeroToken:  true,
			approverVE: veSetFake,
			currentVE:  veSetNone,
			errorMessages: []string{
				"unexpected sender",
				"illegitimate msg",
			},
			errorSeverity: 0,
		},
		{
			name:       "Fail if sender eq approver",
			testRailID: "C5193",
			zeroToken:  false,
			approverVE: veSetServer,
			currentVE:  veSetServer,
			errorMessages: []string{
				"sender cannot be approver of the token",
				"illegitimate msg",
			},
			errorSeverity: throw.FraudSeverity,
		},
		{
			name:       "Fail if wrong approver",
			testRailID: "C5192",
			zeroToken:  false,
			approverVE: veSetFake,
			currentVE:  veSetFake,
			errorMessages: []string{
				"token Approver and expectedVE are different",
				"illegitimate msg",
			},
			errorSeverity: 0,
		},
	}
	messages := []struct {
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
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Log(testCase.testRailID)

			for _, testMsg := range messages {
				mc := minimock.NewController(t)

				server, ctx := utils.NewUninitializedServer(nil, t)

				jetCoordinatorMock := jet.NewAffinityHelperMock(mc)
				auth := authentication.NewService(ctx, jetCoordinatorMock)
				server.ReplaceAuthenticationService(auth)

				var errorFound bool
				logHandler := func(arg interface{}) {
					err, ok := arg.(error)
					if !ok {
						return
					}
					if testCase.errorSeverity > 0 {
						if s, sok := throw.GetSeverity(err); (sok && s != testCase.errorSeverity) || !sok {
							return
						}
					}
					errMsg := err.Error()
					for _, errTemplate := range testCase.errorMessages {
						if !strings.Contains(errMsg, errTemplate) {
							return
						}
					}
					errorFound = true
				}
				logger := utils.InterceptLog(inslogger.FromContext(ctx), logHandler)
				server.OverrideConveyorFactoryLogContext(inslogger.SetLogger(ctx, logger))

				server.Init(ctx)
				// increment pulse for VStateReport and VDelegatedCallRequest
				server.IncrementPulse(ctx)

				switch testCase.approverVE {
				case veSetServer:
					jetCoordinatorMock.
						QueryRoleMock.Return([]reference.Global{server.GlobalCaller()}, nil)
				case veSetFake:
					jetCoordinatorMock.
						QueryRoleMock.Return([]reference.Global{server.RandomGlobalWithPulse()}, nil)
				}

				switch testCase.currentVE {
				case veSetServer:
					jetCoordinatorMock.MeMock.Return(server.GlobalCaller())
				case veSetFake:
					jetCoordinatorMock.MeMock.Return(server.RandomGlobalWithPulse())
				}

				var delegationToken payload.CallDelegationToken
				if testCase.zeroToken {
					delegationToken = payload.CallDelegationToken{
						TokenTypeAndFlags: payload.DelegationTokenTypeUninitialized,
					}
				} else {
					class := gen.UniqueGlobalRef()
					outgoing := server.BuildRandomOutgoingWithPulse()
					delegationToken = server.DelegationToken(reference.NewRecordOf(class, outgoing.GetLocal()), server.GlobalCaller(), outgoing)
				}
				reflect.ValueOf(testMsg.msg).MethodByName("Reset").Call([]reflect.Value{})
				insertToken(delegationToken, testMsg.msg)

				server.SendPayload(ctx, testMsg.msg.(payload.Marshaler))
				server.WaitIdleConveyor()

				assert.True(t, errorFound, "Fail "+testMsg.name)
				server.Stop()
				mc.Finish()
			}
		})
	}
}
