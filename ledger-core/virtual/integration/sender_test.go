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

	"github.com/insolar/assured-ledger/ledger-core/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

var messagesWithoutToken = []struct {
	name  string
	msg   interface{}
	pulse []string
}{
	{
		name:  "VCallRequest",
		msg:   &payload.VCallRequest{},
		pulse: []string{"P"},
	},
	{
		name:  "VCallResult",
		msg:   &payload.VCallResult{},
		pulse: []string{"P"},
	},
	{
		name:  "VStateReport",
		msg:   &payload.VStateReport{},
		pulse: []string{"P-1"},
	},
	{
		name:  "VStateRequest",
		msg:   &payload.VStateRequest{},
		pulse: []string{"P"},
	},
	{
		name:  "VDelegatedCallRequest",
		msg:   &payload.VDelegatedCallRequest{},
		pulse: []string{"P-1"},
	},
	{
		name:  "VDelegatedCallResponse",
		msg:   &payload.VDelegatedCallResponse{},
		pulse: []string{"P"},
	},
	{
		name:  "VDelegatedRequestFinished",
		msg:   &payload.VDelegatedRequestFinished{},
		pulse: []string{"P-1", "any"},
	},
	{
		name:  "VFindCallRequest",
		msg:   &payload.VFindCallRequest{},
		pulse: []string{"P"},
	},
}

func TestSender_SuccessChecks(t *testing.T) {
	t.Log("C5188")
	for _, testMsg := range messagesWithoutToken {
		t.Run(testMsg.name, func(t *testing.T) {
			for _, p := range testMsg.pulse {
				mc := minimock.NewController(t)

				server, ctx := utils.NewUninitializedServerWithErrorFilter(nil, t, func(s string) bool {
					return false
				})

				jetCoordinatorMock := jet.NewAffinityHelperMock(mc)
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
				server.IncrementPulseAndWaitIdle(ctx)

				sender := server.RandomGlobalWithPulse()
				switch p {
				case "P":
				case "P-1":
					server.IncrementPulseAndWaitIdle(ctx)
				case "any":
					server.IncrementPulseAndWaitIdle(ctx)
					server.IncrementPulseAndWaitIdle(ctx)
				default:
					t.Fatal("unexpected")
				}

				reflect.ValueOf(testMsg.msg).MethodByName("Reset").Call([]reflect.Value{})

				logger.Debug("Send message: " + testMsg.name + ", pulse = " + p)
				msg := server.WrapPayload(testMsg.msg.(payload.Marshaler)).SetSender(sender).Finalize()
				server.SendMessage(ctx, msg)

				server.WaitActiveThenIdleConveyor()

				assert.False(t, errorFound, "Fail "+testMsg.name+", pulse = "+p)
				server.Stop()
				mc.Finish()
			}
		})
	}
}

func TestSender_SuccessChecks_VFindCallResponse(t *testing.T) {
	t.Log("C5188")

	mc := minimock.NewController(t)

	server, ctx := utils.NewUninitializedServerWithErrorFilter(nil, t, func(s string) bool {
		return false
	})

	jetCoordinatorMock := jet.NewAffinityHelperMock(mc)
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
	server.IncrementPulseAndWaitIdle(ctx)
	// p1
	p1 := server.GetPulse().PulseNumber
	outgoingP1 := server.BuildRandomOutgoingWithPulse()
	object := reference.NewSelf(server.RandomLocalWithPulse())
	sender := server.RandomGlobalWithPulse()

	// p2
	server.IncrementPulseAndWaitIdle(ctx)

	// send VFindCallRequest
	{
		request := payload.VFindCallRequest{
			LookAt:   p1,
			Callee:   object,
			Outgoing: outgoingP1,
		}
		server.SendPayload(ctx, &request)
	}

	// send VFindCallResponse
	{
		response := payload.VFindCallResponse{
			LookedAt: p1,
			Callee:   object,
			Outgoing: outgoingP1,
			Status:   payload.MissingCall,
		}
		msg := server.WrapPayload(&response).SetSender(sender).Finalize()
		server.SendMessage(ctx, msg)

	}
	server.WaitActiveThenIdleConveyor()

	assert.False(t, errorFound, "Fail ")
	server.Stop()
	mc.Finish()
}
