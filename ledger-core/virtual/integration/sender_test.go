// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"strings"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

var messagesWithoutToken = []struct {
	name              string
	msg               interface{}
	ignoreSenderCheck bool
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
		name: "VDelegatedCallRequest",
		msg:  &payload.VDelegatedCallRequest{},
	},
	{
		name: "VDelegatedCallResponse",
		msg:  &payload.VDelegatedCallResponse{},
	},
	{
		name: "VFindCallRequest",
		msg:  &payload.VFindCallRequest{},
	},
	{
		name:              "VFindCallResponse",
		msg:               &payload.VFindCallResponse{},
		ignoreSenderCheck: true,
	},
}

func ignoreErrors(_ string) bool { return false }

func TestVirtual_SenderCheck_With_ExpectedVE(t *testing.T) {
	defer commontestutils.LeakTester(t)
	testCases := []struct {
		name                    string
		caseId                  string
		senderIsEqualExpectedVE bool
	}{
		{"Sender is equal expectedVE", "C5188", true},
		{"Sender is not equal expectedVE", "C5196", false},
	}

	for _, cases := range testCases {
		t.Run(cases.name, func(t *testing.T) {
			insrail.LogCase(t, cases.caseId)

			for _, testMsg := range messagesWithoutToken {
				t.Run(testMsg.name, func(t *testing.T) {

					mc := minimock.NewController(t)

					server, ctx := utils.NewUninitializedServerWithErrorFilter(nil, t, ignoreErrors)
					defer server.Stop()

					jetCoordinatorMock := affinity.NewHelperMock(mc)
					auth := authentication.NewService(ctx, jetCoordinatorMock)
					server.ReplaceAuthenticationService(auth)
					server.PublisherMock.SetResendMode(ctx, nil)

					var errorFound bool
					{
						logHandler := func(arg interface{}) {
							err, ok := arg.(error)
							if !ok {
								return
							}

							errorMsg := err.Error()
							if strings.Contains(errorMsg, "unexpected sender") &&
								strings.Contains(errorMsg, "illegitimate msg") {
								errorFound = true
							}
						}
						logger := utils.InterceptLog(inslogger.FromContext(ctx), logHandler)
						server.OverrideConveyorFactoryLogContext(inslogger.SetLogger(ctx, logger))
					}

					server.Init(ctx)

					if !testMsg.ignoreSenderCheck {
						rv := server.RandomGlobalWithPulse()
						if cases.senderIsEqualExpectedVE {
							rv = server.GlobalCaller()
						}
						jetCoordinatorMock.QueryRoleMock.Return([]reference.Global{rv}, nil)
					}

					prevPulse := server.GetPrevPulse().PulseNumber

					switch m := (testMsg.msg).(type) {
					case *payload.VStateReport:
						m.Status = payload.StateStatusMissing
						m.AsOf = prevPulse
						m.Object = gen.UniqueGlobalRefWithPulse(prevPulse)

					case *payload.VFindCallRequest:

						m.LookAt = prevPulse
						m.Callee = gen.UniqueGlobalRefWithPulse(prevPulse)
						m.Outgoing = server.BuildRandomOutgoingWithGivenPulse(prevPulse)

					case *payload.VFindCallResponse:

						m.LookedAt = prevPulse
						m.Callee = gen.UniqueGlobalRefWithPulse(prevPulse)
						m.Outgoing = server.BuildRandomOutgoingWithGivenPulse(prevPulse)
						m.Status = payload.CallStateMissing

					case *payload.VDelegatedCallRequest:

						m.Callee = gen.UniqueGlobalRefWithPulse(prevPulse)
						m.CallOutgoing = server.BuildRandomOutgoingWithGivenPulse(prevPulse)
						m.CallIncoming = reference.NewRecordOf(m.Callee, m.CallOutgoing.GetLocal())
						m.CallFlags = payload.CallFlags(0).WithInterference(isolation.CallIntolerable).WithState(isolation.CallValidated)

					case *payload.VDelegatedCallResponse:

						m.Callee = gen.UniqueGlobalRefWithPulse(prevPulse)
						m.CallIncoming = reference.NewRecordOf(m.Callee, gen.UniqueLocalRefWithPulse(prevPulse))

					case *payload.VStateRequest:

						m.AsOf = prevPulse
						m.Object = gen.UniqueGlobalRefWithPulse(prevPulse)

					case *payload.VCallResult:
						m.CallFlags = payload.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty)
						m.CallType = payload.CallTypeMethod
						m.Callee = server.RandomGlobalWithPulse()
						m.Caller = server.GlobalCaller()
						m.CallOutgoing = server.BuildRandomOutgoingWithPulse()
						m.CallIncoming = server.RandomGlobalWithPulse()
						m.ReturnArguments = []byte("some result")

					case *payload.VCallRequest:
						testMsg.msg = utils.GenerateVCallRequestMethod(server)
					}

					server.SendPayload(ctx, testMsg.msg.(payload.Marshaler)) // default caller == server.GlobalCaller()

					expectNoError := cases.senderIsEqualExpectedVE || testMsg.ignoreSenderCheck == true
					if expectNoError {
						// if conveyor got active then we are sure that we passed sender check
						server.WaitActiveThenIdleConveyor()
					} else {
						// we don't wait anything cause Sender check is part of call to SendPayload
						assert.Equal(t, true, errorFound, "Fail "+testMsg.name)
					}

					mc.Finish()
				})
			}
		})
	}
}
