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
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
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
		name: "VDelegatedCallRequest",
		msg:  &rms.VDelegatedCallRequest{},
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
		name:              "VFindCallResponse",
		msg:               &rms.VFindCallResponse{},
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
					case *rms.VStateReport:
						m.Status = rms.StateStatusMissing
						m.AsOf = prevPulse
						m.Object.Set(gen.UniqueGlobalRefWithPulse(prevPulse))

					case *rms.VFindCallRequest:

						m.LookAt = prevPulse
						m.Callee.Set(gen.UniqueGlobalRefWithPulse(prevPulse))
						m.Outgoing.Set(server.BuildRandomOutgoingWithGivenPulse(prevPulse))

					case *rms.VFindCallResponse:

						m.LookedAt = prevPulse
						m.Callee.Set(gen.UniqueGlobalRefWithPulse(prevPulse))
						m.Outgoing.Set(server.BuildRandomOutgoingWithGivenPulse(prevPulse))
						m.Status = rms.CallStateMissing

					case *rms.VDelegatedCallRequest:

						m.Callee.Set(gen.UniqueGlobalRefWithPulse(prevPulse))
						m.CallOutgoing.Set(server.BuildRandomOutgoingWithGivenPulse(prevPulse))
						m.CallIncoming.Set(reference.NewRecordOf(m.Callee.GetValue(), m.CallOutgoing.GetValue().GetLocal()))
						m.CallFlags = rms.CallFlags(0).WithInterference(isolation.CallIntolerable).WithState(isolation.CallValidated)

					case *rms.VDelegatedCallResponse:

						m.Callee.Set(gen.UniqueGlobalRefWithPulse(prevPulse))
						m.CallIncoming.Set(reference.NewRecordOf(m.Callee.GetValue(), gen.UniqueLocalRefWithPulse(prevPulse)))

					case *rms.VStateRequest:

						m.AsOf = prevPulse
						m.Object.Set(gen.UniqueGlobalRefWithPulse(prevPulse))

					case *rms.VCallResult:
						m.CallFlags = rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty)
						m.CallType = rms.CallTypeMethod
						m.Callee.Set(server.RandomGlobalWithPulse())
						m.Caller.Set(server.GlobalCaller())
						m.CallOutgoing.Set(server.BuildRandomOutgoingWithPulse())
						m.CallIncoming.Set(server.RandomGlobalWithPulse())
						m.ReturnArguments.SetBytes([]byte("some result"))

					case *rms.VCallRequest:
						testMsg.msg = utils.GenerateVCallRequestMethod(server)
					}

					server.SendPayload(ctx, testMsg.msg.(rmsreg.GoGoSerializable)) // default caller == server.GlobalCaller()

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
