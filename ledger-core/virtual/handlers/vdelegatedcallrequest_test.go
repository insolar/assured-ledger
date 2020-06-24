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

	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/slotdebugger"
	"github.com/insolar/assured-ledger/ledger-core/virtual/authentication"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
)

var deadBeef = [...]byte{0xde, 0xad, 0xbe, 0xef}

func TestSMVDelegatedCallRequest(t *testing.T) {
	oneRandomOrderedTable := object.NewRequestTable()
	oneRandomOrderedTable.GetList(contract.CallTolerable).Add(gen.UniqueGlobalRef())

	oneRandomUnorderedTable := object.NewRequestTable()
	oneRandomUnorderedTable.GetList(contract.CallIntolerable).Add(gen.UniqueGlobalRef())

	retryOrderedRequestRef := reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow()))
	retryOrderedTable := object.NewRequestTable()
	oneRandomOrderedTable.GetList(contract.CallTolerable).Add(retryOrderedRequestRef)

	retryUnorderedRequestRef := reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow()))
	retryUnorderedTable := object.NewRequestTable()
	oneRandomOrderedTable.GetList(contract.CallTolerable).Add(retryUnorderedRequestRef)

	oneRandomOrderedRequest := reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow()))
	oneRandomUnorderedRequest := reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow()))

	for _, tc := range []struct {
		name                          string
		testRailCase                  string
		requestRef                    reference.Global
		OrderedPendingEarliestPulse   pulse.Number
		UnorderedPendingEarliestPulse pulse.Number
		ActiveOrderedPendingCount     uint8
		ActiveUnorderedPendingCount   uint8
		callFlags                     payload.CallFlags
		expectedResponse              *payload.VDelegatedCallResponse
		PendingRequestTable           object.PendingTable
		expectedError                 bool
	}{
		{
			name:                        "OK tolerable",
			testRailCase:                "C5133",
			PendingRequestTable:         object.NewRequestTable(),
			requestRef:                  oneRandomOrderedRequest,
			OrderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveOrderedPendingCount:   1,
			callFlags:                   payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
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
			testRailCase:                  "C5132",
			PendingRequestTable:           object.NewRequestTable(),
			requestRef:                    oneRandomUnorderedRequest,
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
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
			testRailCase:                "C4987",
			PendingRequestTable:         retryOrderedTable,
			requestRef:                  retryOrderedRequestRef,
			OrderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveOrderedPendingCount:   1,
			callFlags:                   payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
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
			testRailCase:                  "C5127",
			PendingRequestTable:           retryUnorderedTable,
			requestRef:                    retryUnorderedRequestRef,
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
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
			testRailCase:                  "C5131",
			PendingRequestTable:           object.NewRequestTable(),
			requestRef:                    reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow() - 110)),
			UnorderedPendingEarliestPulse: pulse.Unknown,
			ActiveUnorderedPendingCount:   0,
			callFlags:                     payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
			expectedError:                 true,
		},
		{
			name:                          "unexpected tolerable",
			testRailCase:                  "C4985",
			PendingRequestTable:           object.NewRequestTable(),
			requestRef:                    reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow() - 110)),
			UnorderedPendingEarliestPulse: pulse.Unknown,
			ActiveUnorderedPendingCount:   0,
			callFlags:                     payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			expectedError:                 true,
		},
		{
			name:                          "too old intolerable",
			testRailCase:                  "C4984",
			PendingRequestTable:           object.NewRequestTable(),
			requestRef:                    reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow() - 110)),
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
			expectedError:                 true,
		},
		{
			name:                          "too old tolerable",
			testRailCase:                  "C5130",
			PendingRequestTable:           object.NewRequestTable(),
			requestRef:                    reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow() - 110)),
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			expectedError:                 true,
		},
		{
			name:                          "full table intolerable",
			testRailCase:                  "C5129",
			PendingRequestTable:           oneRandomUnorderedTable,
			requestRef:                    reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow() - 110)),
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
			expectedError:                 true,
		},
		{
			name:                          "full table tolerable",
			testRailCase:                  "C4986",
			PendingRequestTable:           oneRandomOrderedTable,
			requestRef:                    reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow() - 110)),
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			expectedError:                 true,
		},
		{
			name:                          "wrong call interference flag: expected intolerable, get tolerable",
			testRailCase:                  "C4989",
			PendingRequestTable:           object.NewRequestTable(),
			requestRef:                    reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow())),
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			expectedError:                 true,
		},
		{
			name:                        "wrong call interference flag: expected tolerable, get intolerable",
			testRailCase:                "C5128",
			PendingRequestTable:         object.NewRequestTable(),
			requestRef:                  reference.NewSelf(gen.UniqueLocalRefWithPulse(pulse.OfNow())),
			OrderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveOrderedPendingCount:   1,
			callFlags:                   payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
			expectedError:               true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Log(tc.testRailCase)
			var (
				mc  = minimock.NewController(t)
				ctx = instestlogger.TestContext(t)

				nodeRef = gen.UniqueGlobalRef()

				caller           = gen.UniqueGlobalRef()
				outgoing         = gen.UniqueLocalRef()
				objectRef        = reference.NewSelf(outgoing)
				orderedBargeIn   = smachine.BargeIn{}
				unorderedBargeIn = smachine.BargeIn{}
				sharedState      = &object.SharedState{
					Info: object.Info{
						PendingTable:                          tc.PendingRequestTable,
						OrderedPendingEarliestPulse:           tc.OrderedPendingEarliestPulse,
						UnorderedPendingEarliestPulse:         tc.UnorderedPendingEarliestPulse,
						PreviousExecutorOrderedPendingCount:   tc.ActiveOrderedPendingCount,
						PreviousExecutorUnorderedPendingCount: tc.ActiveUnorderedPendingCount,
						KnownRequests:                         object.NewWorkingTable(),
						ReadyToWork:                           smsync.NewConditional(1, "ReadyToWork").SyncLink(),
						OrderedExecute:                        smsync.NewConditional(1, "MutableExecution").SyncLink(),
						OrderedPendingListFilledCallback:      orderedBargeIn,
						UnorderedPendingListFilledCallback:    unorderedBargeIn,
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

			affinityHelper := jet.NewAffinityHelperMock(t).MeMock.Return(nodeRef)
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
