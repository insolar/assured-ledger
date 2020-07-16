// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

var messagesWithoutToken = []struct {
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
		name: "VStateRequest",
		msg:  &payload.VStateRequest{},
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
		name: "VStateReport",
		msg:  &payload.VStateReport{},
	},
	{
		name: "VDelegatedCallRequest",
		msg:  &payload.VDelegatedCallRequest{},
	},

	{
		name: "VFindCallResponse",
		msg:  &payload.VFindCallResponse{},
	},
}

func TestSender_SuccessChecks(t *testing.T) {
	testCases := []struct {
		caseId                  string
		senderIsEqualExpectedVE bool
	}{
		{"C5188", true},
		{"C5196", false},
	}

	for _, cases := range testCases {
		t.Run(cases.caseId, func(t *testing.T) {
			t.Log(cases.caseId)
			for _, testMsg := range messagesWithoutToken {
				t.Run(testMsg.name, func(t *testing.T) {

					mc := minimock.NewController(t)

					server, ctx := utils.NewUninitializedServerWithErrorFilter(nil, t, func(s string) bool {
						return false
					})

					jetCoordinatorMock := jet.NewAffinityHelperMock(mc)
					auth := authentication.NewService(ctx, jetCoordinatorMock)
					server.ReplaceAuthenticationService(auth)

					var errorFound bool
					{
						logHandler := func(arg interface{}) {
							err, ok := arg.(error)
							if !ok {
								return
							}
							errMsg := err.Error()
							if strings.Contains(errMsg, "unexpected sender") &&
								strings.Contains(errMsg, "illegitimate msg") {
								errorFound = true
							}
							errorFound = true
						}
						logger := utils.InterceptLog(inslogger.FromContext(ctx), logHandler)
						server.OverrideConveyorFactoryLogContext(inslogger.SetLogger(ctx, logger))
					}

					server.Init(ctx)
					server.IncrementPulseAndWaitIdle(ctx)

					if testMsg.name != "VFindCallResponse" {
						jetCoordinatorMock.QueryRoleMock.Set(func(_ context.Context, _ node.DynamicRole, _ reference.Local, _ pulse.Number) (_ []reference.Global, _ error) {
							if cases.senderIsEqualExpectedVE {
								return []reference.Global{server.GlobalCaller()}, nil
							}
							return []reference.Global{server.RandomGlobalWithPulse()}, nil
						})
					}

					reflect.ValueOf(testMsg.msg).MethodByName("Reset").Call([]reflect.Value{})

					msg := utils.NewRequestWrapper(server.GetPulse().PulseNumber, testMsg.msg.(payload.Marshaler)).SetSender(server.GlobalCaller()).Finalize()
					server.SendMessage(ctx, msg)

					server.WaitIdleConveyor()

					if cases.senderIsEqualExpectedVE || testMsg.name == "VFindCallResponse" {
						assert.False(t, errorFound, "Fail "+testMsg.name)
					} else {
						assert.True(t, errorFound, "Fail "+testMsg.name)
					}

					server.Stop()
					mc.Finish()
				})
			}
		})
	}
}
