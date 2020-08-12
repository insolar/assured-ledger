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
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
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

type reseter interface {
	Reset()
}

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

					server, ctx := utils.NewUninitializedServerWithErrorFilter(nil, t, func(s string) bool {
						return false
					})

					jetCoordinatorMock := affinity.NewHelperMock(mc)
					auth := authentication.NewService(ctx, jetCoordinatorMock)
					server.ReplaceAuthenticationService(auth)

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
					server.IncrementPulseAndWaitIdle(ctx)

					if !testMsg.ignoreSenderCheck {
						rv := server.RandomGlobalWithPulse()
						if cases.senderIsEqualExpectedVE {
							rv = server.GlobalCaller()
						}
						jetCoordinatorMock.QueryRoleMock.Return([]reference.Global{rv}, nil)
					}

					testMsg.msg.(reseter).Reset()

					switch m := (testMsg.msg).(type) {
					case *payload.VStateReport:
						m.Status = payload.Missing
						m.AsOf = server.GetPulse().PulseNumber
						m.Object = reference.NewSelf(server.RandomLocalWithPulse())
						server.IncrementPulseAndWaitIdle(ctx)

						testMsg.msg = m
					case *payload.VFindCallRequest:
						pn := server.GetPulse().PulseNumber
						m.LookAt = pn
						m.Callee = gen.UniqueGlobalRefWithPulse(pn)
						m.Outgoing = gen.UniqueGlobalRefWithPulse(pn)

						testMsg.msg = m
					case *payload.VFindCallResponse:
						m.LookedAt = server.GetPrevPulse().PulseNumber
						m.Status = payload.MissingCall
						m.Callee = reference.NewSelf(gen.UniqueLocalRefWithPulse(m.LookedAt))
						m.Outgoing = reference.New(gen.UniqueLocalRefWithPulse(m.LookedAt), gen.UniqueLocalRefWithPulse(m.LookedAt))
						testMsg.msg = m
					case *payload.VDelegatedCallRequest:
						pn := server.GetPrevPulse().PulseNumber

						m.Callee = gen.UniqueGlobalRefWithPulse(pn)
						m.CallOutgoing = reference.NewRecordOf(server.GlobalCaller(), gen.UniqueLocalRefWithPulse(pn))
						m.CallIncoming = reference.NewRecordOf(m.Callee, m.CallOutgoing.GetLocal())
						m.CallFlags = payload.CallFlags(0).WithInterference(contract.CallIntolerable).WithState(contract.CallValidated)
					case *payload.VDelegatedCallResponse:
						pn := server.GetPrevPulse().PulseNumber

						m.Callee = gen.UniqueGlobalRefWithPulse(pn)
						m.CallIncoming = reference.NewRecordOf(m.Callee, gen.UniqueLocalRefWithPulse(pn))
					case *payload.VStateRequest:
						pn := server.GetPrevPulse().PulseNumber

						m.AsOf = pn
						m.Object = gen.UniqueGlobalRefWithPulse(pn)
					case *payload.VCallResult:
						m.CallFlags = payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty)
						m.CallType = payload.CTMethod
						m.Callee = server.RandomGlobalWithPulse()
						m.Caller = server.GlobalCaller()
						m.CallOutgoing = server.BuildRandomOutgoingWithPulse()
						m.CallIncoming = server.RandomGlobalWithPulse()
						m.ReturnArguments = []byte("some result")
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
					server.Stop()
					mc.Finish()
				})
			}
		})
	}
}
