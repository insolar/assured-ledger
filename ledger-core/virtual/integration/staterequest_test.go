// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

func makeVStateRequestEvent(pulseNumber pulse.Number, ref reference.Global, flags payload.StateRequestContentFlags, sender reference.Global) *message.Message {
	payload := &payload.VStateRequest{
		AsOf:             pulseNumber,
		Object:           ref,
		RequestedContent: flags,
	}

	return utils.NewRequestWrapper(pulseNumber, payload).SetSender(sender).Finalize()
}

func TestVirtual_VStateRequest_WithoutBody(t *testing.T) {
	defer commontestutils.LeakTester(t)

	t.Log("C4861")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()
	server.IncrementPulse(ctx)

	var (
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
		pn           = server.GetPulse().PulseNumber
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	countBefore := server.PublisherMock.GetCount()
	server.IncrementPulse(ctx)
	if !server.PublisherMock.WaitCount(countBefore+1, 10*time.Second) {
		t.Fatal("timeout waiting for VStateReport")
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
		assert.Equal(t, &payload.VStateReport{
			Status:           payload.Ready,
			AsOf:             server.GetPrevPulse().PulseNumber,
			Object:           objectGlobal,
			LatestDirtyState: objectGlobal,
		}, report)

		return false
	})

	countBefore = server.PublisherMock.GetCount()
	msg := makeVStateRequestEvent(pn, objectGlobal, 0, server.JetCoordinatorMock.Me())
	server.SendMessage(ctx, msg)

	if !server.PublisherMock.WaitCount(countBefore+1, 10*time.Second) {
		t.Fatal("timeout waiting for VStateReport")
	}
}

func TestVirtual_VStateRequest_WithBody(t *testing.T) {
	defer commontestutils.LeakTester(t)

	t.Log("C4862")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()
	server.IncrementPulse(ctx)

	var (
		objectLocal    = server.RandomLocalWithPulse()
		objectGlobal   = reference.NewSelf(objectLocal)
		pn             = server.GetPulse().PulseNumber
		rawWalletState = makeRawWalletState(initialBalance)
	)
	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	countBefore := server.PublisherMock.GetCount()
	server.IncrementPulse(ctx)
	if !server.PublisherMock.WaitCount(countBefore+1, 10*time.Second) {
		t.Fatal("timeout waiting for VStateReport")
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
		assert.Equal(t, &payload.VStateReport{
			Status:           payload.Ready,
			AsOf:             pn,
			Object:           objectGlobal,
			LatestDirtyState: objectGlobal,
			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState: &payload.ObjectState{
					Reference: reference.Local{},
					State:     rawWalletState,
					Class:     testwallet.ClassReference,
				},
			},
		}, report)

		return false
	})

	countBefore = server.PublisherMock.GetCount()
	msg := makeVStateRequestEvent(pn, objectGlobal, payload.RequestLatestDirtyState, server.JetCoordinatorMock.Me())
	server.SendMessage(ctx, msg)

	if !server.PublisherMock.WaitCount(countBefore+1, 10*time.Second) {
		t.Fatal("timeout waiting for VStateReport")
	}
}

func TestVirtual_VStateRequest_Unknown(t *testing.T) {
	defer commontestutils.LeakTester(t)

	t.Log("C4863")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	var (
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
		pn           = server.GetPulse().PulseNumber
	)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
		assert.Equal(t, &payload.VStateReport{
			Status: payload.Missing,
			AsOf:   server.GetPrevPulse().PulseNumber,
			Object: objectGlobal,
		}, report)

		return false
	})

	server.IncrementPulse(ctx)

	countBefore := server.PublisherMock.GetCount()
	msg := makeVStateRequestEvent(pn, objectGlobal, payload.RequestLatestDirtyState, server.JetCoordinatorMock.Me())
	server.SendMessage(ctx, msg)

	if !server.PublisherMock.WaitCount(countBefore+1, 10*time.Second) {
		t.Fatal("timeout waiting for VStateReport")
	}
}

func TestVirtual_VStateRequest_WhenObjectIsDeactivated(t *testing.T) {
	t.Log("C5474")
	table := []struct {
		name                         string
		reportStateValidatedIsActive bool
		requestState                 payload.StateRequestContentFlags
	}{
		{name: "Report_ValidatedState = inactive Request_State = dirty",
			reportStateValidatedIsActive: false,
			requestState:                 payload.RequestLatestDirtyState},
		{name: "Report_ValidatedState = inactive Request_State = validated",
			reportStateValidatedIsActive: false,
			requestState:                 payload.RequestLatestValidatedState},
		{name: "Report_ValidatedState=active Request_State = dirty",
			reportStateValidatedIsActive: true,
			requestState:                 payload.RequestLatestDirtyState},
		{name: "Report_ValidatedState=active Request_State = validated",
			reportStateValidatedIsActive: true,
			requestState:                 payload.RequestLatestValidatedState},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			mc := minimock.NewController(t)

			server, ctx := utils.NewServerWithErrorFilter(nil, t, func(s string) bool {
				return !strings.Contains(s, "(*SMExecute).stepSaveNewObject") // todo
			})
			defer server.Stop()

			var (
				class        = testwallet.GetClass()
				objectGlobal = reference.NewSelf(server.RandomLocalWithPulse())

				dirtyStateRef     = server.RandomLocalWithPulse()
				validatedStateRef = server.RandomLocalWithPulse()
				pulseNumber       = server.GetPulse().PulseNumber
			)

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			// todo

			// Send VStateReport with Dirty, Validated states
			{
				validatedState := payload.Inactive
				if test.reportStateValidatedIsActive {
					validatedState = payload.Ready
				}
				dirtyState := payload.Inactive

				content := &payload.VStateReport_ProvidedContentBody{
					LatestDirtyState: &payload.ObjectState{
						Reference: dirtyStateRef,
						Class:     class,
						State:     insolar.MustSerialize(dirtyState),
					},
					LatestValidatedState: &payload.ObjectState{
						Reference: validatedStateRef,
						Class:     class,
						State:     insolar.MustSerialize(validatedState),
					},
				}

				pl := &payload.VStateReport{
					AsOf:            pulseNumber,
					Status:          payload.Inactive,
					Object:          objectGlobal,
					ProvidedContent: content,
				}
				server.SendPayload(ctx, pl)
				testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1))
			}

			server.IncrementPulse(ctx)

			// VStateRequest
			{
				payload := &payload.VStateRequest{
					AsOf:             pulseNumber,
					Object:           objectGlobal,
					RequestedContent: test.requestState,
				}
				server.SendPayload(ctx, payload)
			}

			testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			assert.Equal(t, 1, typedChecker.VStateReport.Count())
		})
	}
}

func TestVirtual_CallMethod_WhenObjectIsDeactivated(t *testing.T) {
	t.Log("C5474") // todo
	table := []struct {
		name          string
		requestState  contract.StateFlag
		expectedError bool
	}{
		{name: "Request_State = dirty",
			requestState:  contract.CallDirty,
			expectedError: true},
		{name: "Request_State = validated",
			requestState:  contract.CallValidated,
			expectedError: false,
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			mc := minimock.NewController(t)

			server, ctx := utils.NewServerWithErrorFilter(nil, t, func(s string) bool {
				return !strings.Contains(s, "(*SMExecute).stepSaveNewObject")
			})
			defer server.Stop()

			var (
				class        = testwallet.GetClass()
				objectGlobal = reference.NewSelf(server.RandomLocalWithPulse())

				dirtyStateRef     = server.RandomLocalWithPulse()
				validatedStateRef = server.RandomLocalWithPulse()
				pulseNumber       = server.GetPulse().PulseNumber
			)

			typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
			typedChecker.VCallResult.Set(func(result *payload.VCallResult) bool {

				return false
			})
			// todo

			// Send VStateReport with Dirty, Validated states
			{
				validatedState := payload.Ready
				dirtyState := payload.Inactive

				content := &payload.VStateReport_ProvidedContentBody{
					LatestDirtyState: &payload.ObjectState{
						Reference: dirtyStateRef,
						Class:     class,
						State:     insolar.MustSerialize(dirtyState),
					},
					LatestValidatedState: &payload.ObjectState{
						Reference: validatedStateRef,
						Class:     class,
						State:     insolar.MustSerialize(validatedState),
					},
				}

				pl := &payload.VStateReport{
					AsOf:            pulseNumber,
					Status:          payload.Inactive,
					Object:          objectGlobal,
					ProvidedContent: content,
				}
				server.SendPayload(ctx, pl)
				testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1))
			}

			server.IncrementPulse(ctx)

			{
				payload := &payload.VCallRequest{
					CallType:            payload.CTMethod,
					CallFlags:           payload.BuildCallFlags(contract.CallIntolerable, test.requestState),
					Caller:              server.GlobalCaller(),
					Callee:              objectGlobal,
					CallSiteDeclaration: class,
					CallSiteMethod:      "GetBalance",
					CallOutgoing:        server.BuildRandomOutgoingWithPulse(),
				}
				server.SendPayload(ctx, payload)
			}

			testutils.WaitSignalsTimed(t, 10*time.Second, server.Journal.WaitAllAsyncCallsDone())

			assert.Equal(t, 1, typedChecker.VCallResult.Count())
		})
	}
}
