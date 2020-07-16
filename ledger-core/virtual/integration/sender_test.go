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
	name     string
	msg      interface{}
	sendFrom string
}{
	{
		name:     "VCallRequest",
		msg:      &payload.VCallRequest{},
		sendFrom: "P",
	},
	{
		name:     "VCallResult",
		msg:      &payload.VCallResult{},
		sendFrom: "P",
	},
	{
		name:     "VStateRequest",
		msg:      &payload.VStateRequest{},
		sendFrom: "P",
	},
	{
		name:     "VDelegatedCallResponse",
		msg:      &payload.VDelegatedCallResponse{},
		sendFrom: "P",
	},
	{
		name:     "VFindCallRequest",
		msg:      &payload.VFindCallRequest{},
		sendFrom: "P",
	},

	{
		name:     "VStateReport",
		msg:      &payload.VStateReport{},
		sendFrom: "P-1",
	},
	{
		name:     "VDelegatedCallRequest",
		msg:      &payload.VDelegatedCallRequest{},
		sendFrom: "P-1",
	},

	{
		name:     "VFindCallResponse",
		msg:      &payload.VFindCallResponse{},
		sendFrom: "P-N",
	},
}

func TestSender_SuccessChecks(t *testing.T) {
	t.Log("C5188")
	for _, testMsg := range messagesWithoutToken {
		t.Run(testMsg.name, func(t *testing.T) {
			trueSender := []bool{true, false}
			for _, s := range trueSender {

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

				// for sendFrom == P
				sender := server.RandomGlobalWithPulse()
				msgPulse := server.GetPulse().PulseNumber

				if testMsg.name != "VFindCallResponse" {
					jetCoordinatorMock.QueryRoleMock.Set(func(ctx context.Context, role node.DynamicRole, obj reference.Local, pulse pulse.Number) (ga1 []reference.Global, err error) {
						if s {
							return []reference.Global{server.GlobalCaller()}, nil
						}
						return []reference.Global{sender}, nil
					})
				}

				if testMsg.sendFrom == "P-1" {
					server.IncrementPulse(ctx)
				} else if testMsg.sendFrom == "P-N" {
					server.IncrementPulse(ctx)
					server.IncrementPulse(ctx)
					server.IncrementPulse(ctx)
				}

				reflect.ValueOf(testMsg.msg).MethodByName("Reset").Call([]reflect.Value{})

				msg := utils.NewRequestWrapper(msgPulse, testMsg.msg.(payload.Marshaler)).SetSender(server.GlobalCaller()).Finalize()
				server.SendMessage(ctx, msg)

				server.WaitActiveThenIdleConveyor()

				if s {
					assert.False(t, errorFound, "Fail "+testMsg.name)
				} else {
					assert.True(t, errorFound, "Fail "+testMsg.name)
				}

				server.Stop()
				mc.Finish()
			}
		})

	}
}
