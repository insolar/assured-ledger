// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handlers

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/appctl/affinity"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/testutils/slotdebugger"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/callregistry"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

var deadBeef = [...]byte{0xde, 0xad, 0xbe, 0xef}

func TestSMVDelegatedCallRequest(t *testing.T) {
	insrail.LogCase(t, "C4984")
	oneRandomOrderedTable := callregistry.NewRequestTable()
	oneRandomOrderedTable.GetList(isolation.CallTolerable).Add(gen.UniqueGlobalRef())

	oneRandomUnorderedTable := callregistry.NewRequestTable()
	oneRandomUnorderedTable.GetList(isolation.CallIntolerable).Add(gen.UniqueGlobalRef())

	retryOrderedRequestRef := reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow()))
	retryOrderedTable := callregistry.NewRequestTable()
	oneRandomOrderedTable.GetList(isolation.CallTolerable).Add(retryOrderedRequestRef)

	retryUnorderedRequestRef := reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow()))
	retryUnorderedTable := callregistry.NewRequestTable()
	oneRandomOrderedTable.GetList(isolation.CallTolerable).Add(retryUnorderedRequestRef)

	oneRandomOrderedRequest := reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow()))
	oneRandomUnorderedRequest := reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow()))

	for _, tc := range []struct {
		name                          string
		requestRef                    reference.Global
		OrderedPendingEarliestPulse   pulse.Number
		UnorderedPendingEarliestPulse pulse.Number
		ActiveOrderedPendingCount     uint8
		ActiveUnorderedPendingCount   uint8
		callFlags                     payload.CallFlags
		expectedResponse              *payload.VDelegatedCallResponse
		PendingRequestTable           callregistry.PendingTable
		expectedError                 bool
	}{
		{
			name:                        "OK tolerable",
			PendingRequestTable:         callregistry.NewRequestTable(),
			requestRef:                  oneRandomOrderedRequest,
			OrderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveOrderedPendingCount:   1,
			callFlags:                   payload.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
			expectedResponse: &payload.VDelegatedCallResponse{
				ResponseDelegationSpec: payload.CallDelegationToken{
					TokenTypeAndFlags: payload.DelegationTokenTypeCall,
					ApproverSignature: deadBeef[:],
					Outgoing:          oneRandomOrderedRequest,
				},
			},
		},
		{
			name:                          "OK intolerable",
			PendingRequestTable:           callregistry.NewRequestTable(),
			requestRef:                    oneRandomUnorderedRequest,
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
			expectedResponse: &payload.VDelegatedCallResponse{
				ResponseDelegationSpec: payload.CallDelegationToken{
					TokenTypeAndFlags: payload.DelegationTokenTypeCall,
					ApproverSignature: deadBeef[:],
					Outgoing:          oneRandomUnorderedRequest,
				},
			},
		},
		{
			name:                        "retry tolerable",
			PendingRequestTable:         retryOrderedTable,
			requestRef:                  retryOrderedRequestRef,
			OrderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveOrderedPendingCount:   1,
			callFlags:                   payload.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
			expectedResponse: &payload.VDelegatedCallResponse{
				ResponseDelegationSpec: payload.CallDelegationToken{
					TokenTypeAndFlags: payload.DelegationTokenTypeCall,
					ApproverSignature: deadBeef[:],
					Outgoing:          retryOrderedRequestRef,
				},
			},
		},
		{
			name:                          "retry intolerable",
			PendingRequestTable:           retryUnorderedTable,
			requestRef:                    retryUnorderedRequestRef,
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
			expectedResponse: &payload.VDelegatedCallResponse{
				ResponseDelegationSpec: payload.CallDelegationToken{
					TokenTypeAndFlags: payload.DelegationTokenTypeCall,
					PulseNumber:       pulse.OfNow(),
					ApproverSignature: deadBeef[:],
					Outgoing:          retryUnorderedRequestRef,
				},
			},
		},
		{
			name:                          "unexpected intolerable",
			PendingRequestTable:           callregistry.NewRequestTable(),
			requestRef:                    reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow() - 110)),
			UnorderedPendingEarliestPulse: pulse.Unknown,
			ActiveUnorderedPendingCount:   0,
			callFlags:                     payload.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
			expectedError:                 true,
		},
		{
			name:                        "unexpected tolerable",
			PendingRequestTable:         callregistry.NewRequestTable(),
			requestRef:                  reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow() - 110)),
			OrderedPendingEarliestPulse: pulse.Unknown,
			ActiveOrderedPendingCount:   0,
			callFlags:                   payload.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
			expectedError:               true,
		},
		{
			name:                          "too old intolerable",
			PendingRequestTable:           callregistry.NewRequestTable(),
			requestRef:                    reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow() - 110)),
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
			expectedError:                 true,
		},
		{
			name:                        "too old tolerable",
			PendingRequestTable:         callregistry.NewRequestTable(),
			requestRef:                  reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow() - 110)),
			OrderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveOrderedPendingCount:   1,
			callFlags:                   payload.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
			expectedError:               true,
		},
		{
			name:                          "full table intolerable",
			PendingRequestTable:           oneRandomUnorderedTable,
			requestRef:                    reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow() - 110)),
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
			expectedError:                 true,
		},
		{
			name:                        "full table tolerable",
			PendingRequestTable:         oneRandomOrderedTable,
			requestRef:                  reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow() - 110)),
			OrderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveOrderedPendingCount:   1,
			callFlags:                   payload.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
			expectedError:               true,
		},
		{
			name:                          "wrong call interference flag_expected intolerable, get tolerable",
			PendingRequestTable:           callregistry.NewRequestTable(),
			requestRef:                    reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow())),
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
			expectedError:                 true,
		},
		{
			name:                        "wrong call interference flag_expected tolerable, get intolerable",
			PendingRequestTable:         callregistry.NewRequestTable(),
			requestRef:                  reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow())),
			OrderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveOrderedPendingCount:   1,
			callFlags:                   payload.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
			expectedError:               true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer commontestutils.LeakTester(t)

			var (
				mc  = minimock.NewController(t)
				ctx = instestlogger.TestContext(t)

				nodeRef = gen.UniqueGlobalRef()

				caller      = gen.UniqueGlobalRef()
				outgoing    = gen.UniqueLocalRef()
				objectRef   = reference.NewSelf(outgoing)
				sharedState = &object.SharedState{
					Info: object.Info{
						PendingTable:                          tc.PendingRequestTable,
						OrderedPendingEarliestPulse:           tc.OrderedPendingEarliestPulse,
						UnorderedPendingEarliestPulse:         tc.UnorderedPendingEarliestPulse,
						PreviousExecutorOrderedPendingCount:   tc.ActiveOrderedPendingCount,
						PreviousExecutorUnorderedPendingCount: tc.ActiveUnorderedPendingCount,
						KnownRequests:                         callregistry.NewWorkingTable(),
						ReadyToWork:                           smsync.NewConditional(1, "ReadyToWork").SyncLink(),
						OrderedExecute:                        smsync.NewConditional(1, "MutableExecution").SyncLink(),
					},
				}
				callFlags = tc.callFlags
			)

			sharedState.SetState(object.HasState)

			var slotMachine *slotdebugger.StepController
			if tc.expectedError {
				slotMachine = slotdebugger.NewWithIgnoreAllErrors(ctx, t)
				slotMachine.InitEmptyMessageSender(mc)
			} else {
				slotMachine = slotdebugger.New(ctx, t)
				slotMachine.PrepareMockedMessageSender(mc)
			}

			affinityHelper := affinity.NewHelperMock(t).MeMock.Return(nodeRef)
			var authenticationService = authentication.NewService(ctx, affinityHelper)

			slotMachine.AddInterfaceDependency(&authenticationService)

			smDelegatedCallRequest := SMVDelegatedCallRequest{
				Payload: &payload.VDelegatedCallRequest{
					CallOutgoing: tc.requestRef,
					CallFlags:    callFlags,
					Callee:       objectRef,
				},
				Meta: &payload.Meta{
					Sender: caller,
				},
			}
			catalogWrapper := object.NewCatalogMockWrapper(mc)

			{
				var catalog object.Catalog = catalogWrapper.Mock()
				slotMachine.AddInterfaceDependency(&catalog)

				sharedStateData := smachine.NewUnboundSharedData(sharedState)
				smObjectAccessor := object.SharedStateAccessor{SharedDataLink: sharedStateData}

				catalogWrapper.AddObject(objectRef, smObjectAccessor)
				catalogWrapper.AllowAccessMode(object.CatalogMockAccessGetOrCreate)
			}

			slotMachine.Start()
			defer slotMachine.Stop()

			if tc.expectedError {

				smWrapper := slotMachine.AddStateMachine(ctx, &smDelegatedCallRequest)

				slotMachine.RunTil(smWrapper.AfterStop())
				mc.Finish()

				return
			}

			expectedResponse := tc.expectedResponse
			expectedResponse.ResponseDelegationSpec.Approver = nodeRef
			expectedResponse.ResponseDelegationSpec.DelegateTo = caller
			expectedResponse.ResponseDelegationSpec.Caller = caller
			expectedResponse.ResponseDelegationSpec.PulseNumber = pulse.OfNow()
			expectedResponse.ResponseDelegationSpec.Callee = objectRef
			expectedResponse.Callee = objectRef

			slotMachine.MessageSender.SendTarget.Set(func(_ context.Context, msg payload.Marshaler, target reference.Global, _ ...messagesender.SendOption) error {
				res := msg.(*payload.VDelegatedCallResponse)
				// ensure that both times request is the same
				require.Equal(t, caller, target)
				require.Equal(t, expectedResponse, res)
				return nil
			})

			smWrapper := slotMachine.AddStateMachine(ctx, &smDelegatedCallRequest)

			slotMachine.RunTil(smWrapper.AfterStep(smDelegatedCallRequest.stepBuildResponse))

			require.True(t, sharedState.PendingTable.GetList(callFlags.GetInterference()).Exist(tc.requestRef))

			require.NoError(t, catalogWrapper.CheckDone())
			mc.Finish()
		})
	}
}
