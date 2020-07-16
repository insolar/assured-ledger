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

	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

var messagesWithoutToken = []struct {
	name        string
	msg         interface{}
	validPulses []string
}{
	{
		name:        "VCallRequest",
		msg:         &payload.VCallRequest{},
		validPulses: []string{"P"},
	},
	{
		name:        "VCallResult",
		msg:         &payload.VCallResult{},
		validPulses: []string{"P"},
	},
	{
		name:        "VStateReport",
		msg:         &payload.VStateReport{},
		validPulses: []string{"P-1"},
	},
	{
		name:        "VStateRequest",
		msg:         &payload.VStateRequest{},
		validPulses: []string{"P"},
	},
	{
		name:        "VDelegatedCallRequest",
		msg:         &payload.VDelegatedCallRequest{},
		validPulses: []string{"P-1"},
	},
	{
		name:        "VDelegatedCallResponse",
		msg:         &payload.VDelegatedCallResponse{},
		validPulses: []string{"P"},
	},
	{
		name:        "VDelegatedRequestFinished",
		msg:         &payload.VDelegatedRequestFinished{},
		validPulses: []string{"P-1", "any"},
	},
	{
		name:        "VFindCallRequest",
		msg:         &payload.VFindCallRequest{},
		validPulses: []string{"P"},
	},
	{
		name:        "VFindCallResponse",
		msg:         &payload.VFindCallResponse{},
		validPulses: []string{"P-1", "any"},
	},
}

func TestSender_SuccessChecks(t *testing.T) {
	t.Log("C5188")
	for _, testMsg := range messagesWithoutToken {
		t.Run(testMsg.name, func(t *testing.T) {
			for _, p := range []string{"P", "P-1", "any"} {
				t.Run(p, func(t *testing.T) {
					mc := minimock.NewController(t)

					server, ctx := utils.NewUninitializedServerWithErrorFilter(nil, t, func(s string) bool {
						return false
					})

					var errorFound bool
					logHandler := func(arg interface{}) {
						err, ok := arg.(error)
						if !ok {
							return
						}
						errMsg := err.Error()
						if strings.Contains(errMsg, "unexpected sender") ||
							strings.Contains(errMsg, "illegitimate msg") ||
							strings.Contains(errMsg, "rejected msg") {
							errorFound = true
						}
						errorFound = true
					}
					logger := utils.InterceptLog(inslogger.FromContext(ctx), logHandler)
					server.OverrideConveyorFactoryLogContext(inslogger.SetLogger(ctx, logger))

					server.Init(ctx)
					server.IncrementPulseAndWaitIdle(ctx)

					sender := server.GlobalCaller()
					msgPulse := server.GetPulse().PulseNumber

					switch p {
					case "P":
					case "P-1":
						server.IncrementPulse(ctx)
					case "any":
						server.IncrementPulse(ctx)
						server.IncrementPulse(ctx)
					default:
						t.Fatal("unexpected")
					}

					reflect.ValueOf(testMsg.msg).MethodByName("Reset").Call([]reflect.Value{})

					logger.Debug("Send message: " + testMsg.name + ", validPulses = " + p)
					msg := utils.NewRequestWrapper(msgPulse, testMsg.msg.(payload.Marshaler)).SetSender(sender).Finalize()
					server.SendMessage(ctx, msg)

					server.WaitActiveThenIdleConveyor()

					sliceContains := func(s []string, e string) bool {
						for _, a := range s {
							if a == e {
								return true
							}
						}
						return false
					}
					if sliceContains(testMsg.validPulses, p) {
						assert.False(t, errorFound, "Fail "+testMsg.name+", pulse = "+p)
					} else {
						assert.True(t, errorFound, "Fail "+testMsg.name+", pulse = "+p)
					}
					server.Stop()
					mc.Finish()
				})
			}
		})
	}
}
