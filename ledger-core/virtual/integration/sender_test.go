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
					defer commontestutils.LeakTester(t)

					mc := minimock.NewController(t)

					server, ctx := utils.NewUninitializedServerWithErrorFilter(nil, t, func(s string) bool {
						return false
					})

					jetCoordinatorMock := affinity.NewHelperMock(mc)
					auth := authentication.NewService(ctx, jetCoordinatorMock)
					server.ReplaceAuthenticationService(auth)

					var (
						unexpectedError error
						errorFound      bool
					)
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
							} else {
								unexpectedError = err
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
						pulse := server.GetPulse().PulseNumber
						m.LookAt = pulse
						m.Callee = gen.UniqueGlobalRefWithPulse(pulse)
						m.Outgoing = gen.UniqueGlobalRefWithPulse(pulse)

						testMsg.msg = m
					case *payload.VFindCallResponse:
						m.LookedAt = server.GetPrevPulse().PulseNumber
						m.Status = payload.MissingCall
						m.Callee = reference.NewSelf(gen.UniqueLocalRefWithPulse(m.LookedAt))
						m.Outgoing = reference.New(gen.UniqueLocalRefWithPulse(m.LookedAt), gen.UniqueLocalRefWithPulse(m.LookedAt))
					}

					server.SendPayload(ctx, testMsg.msg.(payload.Marshaler)) // default caller == server.GlobalCaller()

					server.WaitIdleConveyor()

					expectNoError := cases.senderIsEqualExpectedVE || testMsg.ignoreSenderCheck == true
					assert.Equal(t, !expectNoError, errorFound, "Fail "+testMsg.name)
					assert.NoError(t, unexpectedError)

					server.Stop()
					mc.Finish()
				})
			}
		})
	}
}
