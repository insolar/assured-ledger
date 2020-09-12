// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

var messagesWithToken = []struct {
	name string
	msg  interface{}
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
		name: "VStateReport",
		msg:  &rms.VStateReport{},
	},
	{
		name: "VStateRequest",
		msg:  &rms.VStateRequest{},
	},
	{
		name: "VDelegatedCallRequest",
		msg:  &rms.VDelegatedCallRequest{},
	},
	{
		name: "VDelegatedRequestFinished",
		msg:  &rms.VDelegatedRequestFinished{},
	},
}

func TestDelegationToken_SuccessCheckCorrectToken(t *testing.T) {
	insrail.LogCase(t, "C5191")

	for _, testMsg := range messagesWithToken {
		t.Run(testMsg.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServerWithErrorFilter(nil, t, func(s string) bool {
				return false
			})

			jetCoordinatorMock := affinity.NewHelperMock(mc)
			auth := authentication.NewService(ctx, jetCoordinatorMock)
			server.ReplaceAuthenticationService(auth)

			var errorFound bool
			logHandler := func(arg interface{}) {
				err, ok := arg.(error)
				if !ok {
					return
				}
				errMsg := err.Error()
				if !strings.Contains(errMsg, "illegitimate msg") {
					return
				}
				errorFound = true
			}
			logger := utils.InterceptLog(inslogger.FromContext(ctx), logHandler)
			server.OverrideConveyorFactoryLogContext(inslogger.SetLogger(ctx, logger))

			server.Init(ctx)
			// increment pulse for VStateReport and VDelegatedCallRequest
			server.IncrementPulse(ctx)

			approver := server.RandomGlobalWithPulse()
			jetCoordinatorMock.
				MeMock.Return(approver).
				QueryRoleMock.Return([]reference.Global{approver}, nil)

			class := server.RandomGlobalWithPulse()
			outgoing := server.BuildRandomOutgoingWithPulse()
			delegationToken := server.DelegationToken(reference.NewRecordOf(class, outgoing.GetLocal()), server.GlobalCaller(), outgoing)

			reflect.ValueOf(testMsg.msg).MethodByName("Reset").Call([]reflect.Value{})
			insertToken(delegationToken, testMsg.msg)

			server.SendPayload(ctx, testMsg.msg.(rmsreg.GoGoSerializable))
			server.WaitActiveThenIdleConveyor()

			assert.False(t, errorFound)

			server.Stop()
			mc.Finish()
		})
	}
}

func TestDelegationToken_CheckTokenField(t *testing.T) {
	insrail.LogSkipCase(t, "C5197", "https://insolar.atlassian.net/browse/PLAT-588")

	tests := []struct {
		name         string
		fakeCaller   bool
		fakeCallee   bool
		fakeOutgoing bool
	}{
		{
			name:       "Fail with wrong caller in token",
			fakeCaller: true,
		},
		{
			name:       "Fail with wrong callee in token",
			fakeCallee: true,
		},
		{
			name:         "Fail with wrong outgoing in token",
			fakeOutgoing: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			jetCoordinatorMock := affinity.NewHelperMock(t)
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

			pl := utils.GenerateVCallRequestConstructor(server)

			var (
				delegationToken rms.CallDelegationToken
			)

			delegationToken = server.DelegationToken(pl.CallOutgoing.GetValue(), pl.Caller.GetValue(), pl.Callee.GetValue())
			switch {
			case test.fakeCaller:
				delegationToken.Caller.Set(server.RandomGlobalWithPulse())
			case test.fakeCallee:
				delegationToken.Callee.Set(server.RandomGlobalWithPulse())
			case test.fakeOutgoing:
				delegationToken.Outgoing.Set(server.RandomGlobalWithPulse())
			}

			pl.DelegationSpec = delegationToken

			server.SendPayload(ctx, pl)
			server.WaitIdleConveyor()

			assert.True(t, errorFound)

			mc.Finish()
		})
	}
}

func insertToken(token rms.CallDelegationToken, msg interface{}) {
	field := reflect.New(reflect.TypeOf(token))
	field.Elem().Set(reflect.ValueOf(token))
	reflect.ValueOf(msg).Elem().FieldByName("DelegationSpec").Set(field.Elem())
}

type veSetMode int

const (
	veSetServer veSetMode = iota
	veSetFake
	veSetFixed
)

type testCase struct {
	name           string
	testRailID     string
	zeroToken      bool
	approverVE     veSetMode
	expectedVE     veSetMode
	errorMessages  []string
	errorSeverity  throw.Severity
	customDelegate reference.Global
}

func TestDelegationToken_CheckMessageFromAuthorizedVirtual(t *testing.T) {
	insrail.LogCase(t, "C5192")

	fixedVe := gen.UniqueGlobalRef()
	cases := []testCase{
		{
			name:       "Fail if sender eq approver",
			zeroToken:  false,
			expectedVE: veSetServer,
			approverVE: veSetServer,
			errorMessages: []string{
				"sender cannot be approver of the token",
				"illegitimate msg",
			},
			errorSeverity: throw.FraudSeverity,
		},
		{
			name:       "Fail if wrong approver",
			zeroToken:  false,
			expectedVE: veSetFake,
			approverVE: veSetFake,
			errorMessages: []string{
				"token Approver and expectedVE are different",
				"illegitimate msg",
			},
			errorSeverity: 0,
		},
		{
			name:       "Fail if wrong delegate",
			zeroToken:  false,
			expectedVE: veSetFixed,
			approverVE: veSetFixed,
			errorMessages: []string{
				"token DelegateTo and sender are different",
				"illegitimate msg",
			},
			errorSeverity:  throw.RemoteBreachSeverity,
			customDelegate: gen.UniqueGlobalRef(),
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			for _, testMsg := range messagesWithToken {
				mc := minimock.NewController(t)

				server, ctx := utils.NewUninitializedServer(nil, t)

				jetCoordinatorMock := affinity.NewHelperMock(mc)
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

				switch testCase.expectedVE {
				case veSetServer:
					jetCoordinatorMock.
						QueryRoleMock.Return([]reference.Global{server.GlobalCaller()}, nil)
				case veSetFake:
					jetCoordinatorMock.
						QueryRoleMock.Return([]reference.Global{server.RandomGlobalWithPulse()}, nil)
				case veSetFixed:
					jetCoordinatorMock.
						QueryRoleMock.Return([]reference.Global{fixedVe}, nil)
				}

				switch testCase.approverVE {
				case veSetServer:
					jetCoordinatorMock.MeMock.Return(server.GlobalCaller())
				case veSetFake:
					jetCoordinatorMock.MeMock.Return(server.RandomGlobalWithPulse())
				case veSetFixed:
					jetCoordinatorMock.MeMock.Return(fixedVe)
				}

				var delegationToken rms.CallDelegationToken
				if testCase.zeroToken {
					delegationToken = rms.CallDelegationToken{
						TokenTypeAndFlags: rms.DelegationTokenTypeUninitialized,
					}
				} else {
					class := server.RandomGlobalWithPulse()
					outgoing := server.BuildRandomOutgoingWithPulse()
					delegateTo := server.GlobalCaller()
					if !testCase.customDelegate.IsZero() {
						delegateTo = testCase.customDelegate
					}
					delegationToken = server.DelegationToken(reference.NewRecordOf(class, outgoing.GetLocal()), delegateTo, outgoing)
				}
				reflect.ValueOf(testMsg.msg).MethodByName("Reset").Call([]reflect.Value{})
				insertToken(delegationToken, testMsg.msg)

				server.SendPayload(ctx, testMsg.msg.(rmsreg.GoGoSerializable))
				server.WaitIdleConveyor()

				assert.True(t, errorFound, "Fail "+testMsg.name)
				server.Stop()
				mc.Finish()
			}
		})
	}
}

func TestDelegationToken_OldVEVDelegatedCallRequest(t *testing.T) {
	insrail.LogCase(t, "C5186")

	testCases := []struct {
		name          string
		haveCorrectDT bool
	}{
		{
			name:          "Success run SM if DT is correct",
			haveCorrectDT: true,
		},
		{
			name:          "Fail if message have no DT",
			haveCorrectDT: false,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			mc := minimock.NewController(t)

			server, ctx := utils.NewUninitializedServer(nil, t)
			defer server.Stop()

			executeDone := server.Journal.WaitStopOf(&handlers.SMVDelegatedCallRequest{}, 1)

			jetCoordinatorMock := affinity.NewHelperMock(t)
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

			var (
				object      = server.RandomGlobalWithPulse()
				outgoing    = server.BuildRandomOutgoingWithPulse()
				incoming    = reference.NewRecordOf(object, outgoing.GetLocal())
				executorRef = server.RandomGlobalWithPulse()
				firstPulse  = server.GetPulse()

				delegationToken, expectedToken      rms.CallDelegationToken
				expectedVDelegatedCallResponseCount int
			)

			server.IncrementPulse(ctx)

			p := server.GetPulse().PulseNumber

			approver := server.RandomGlobalWithPulse()
			jetCoordinatorMock.
				MeMock.Return(approver).
				QueryRoleMock.Return([]reference.Global{approver}, nil)

			if test.haveCorrectDT {
				delegationToken = server.DelegationToken(outgoing, executorRef, object)
			}

			server.IncrementPulse(ctx)

			if test.haveCorrectDT {
				expectedToken = server.DelegationToken(outgoing, executorRef, object)
				assert.NotEqual(t, delegationToken.PulseNumber, expectedToken.PulseNumber)
				assert.Equal(t, delegationToken.DelegateTo, expectedToken.DelegateTo)
			}

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			typedChecker.VDelegatedCallResponse.Set(func(response *rms.VDelegatedCallResponse) bool {
				assert.Equal(t, object, response.Callee.GetValue())
				utils.AssertCallDelegationTokenEqual(t, &expectedToken, &response.ResponseDelegationSpec)
				return false
			})

			statePl := rms.VStateReport{
				Status:                      rms.StateStatusEmpty,
				Object:                      rms.NewReference(object),
				AsOf:                        p,
				OrderedPendingCount:         1,
				OrderedPendingEarliestPulse: firstPulse.PulseNumber,
			}
			server.SendMessage(ctx, utils.NewRequestWrapper(server.GetPulse().PulseNumber, &statePl).SetSender(approver).Finalize())
			server.WaitActiveThenIdleConveyor()

			pl := rms.VDelegatedCallRequest{
				Callee:         rms.NewReference(object),
				CallFlags:      rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
				CallIncoming:   rms.NewReference(incoming),
				CallOutgoing:   rms.NewReference(outgoing),
				DelegationSpec: delegationToken,
			}
			msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, &pl).SetSender(executorRef).Finalize()
			server.SendMessage(ctx, msg)

			if test.haveCorrectDT {
				commontestutils.WaitSignalsTimed(t, 10*time.Second, executeDone)
				commontestutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())
				expectedVDelegatedCallResponseCount = 1
			}

			assert.Equal(t, !test.haveCorrectDT, errorFound)
			assert.Equal(t, expectedVDelegatedCallResponseCount, typedChecker.VDelegatedCallResponse.Count())

			mc.Finish()
		})
	}
}
