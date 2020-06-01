// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handlers

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine"
	"github.com/insolar/assured-ledger/ledger-core/v2/conveyor/smachine/smsync"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/jet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils/slotdebugger"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/object"
)

func TestSMVDelegatedCallRequest(t *testing.T) {
	oneRandomOrderedTable := object.NewRequestTable()
	oneRandomOrderedTable.GetList(contract.CallTolerable).Add(gen.UniqueReference())

	oneRandomUnorderedTable := object.NewRequestTable()
	oneRandomUnorderedTable.GetList(contract.CallIntolerable).Add(gen.UniqueReference())

	retryOrderedRequestRef := reference.NewSelf(gen.UniqueIDWithPulse(pulse.OfNow()))
	retryOrderedTable := object.NewRequestTable()
	oneRandomOrderedTable.GetList(contract.CallTolerable).Add(retryOrderedRequestRef)

	retryUnorderedRequestRef := reference.NewSelf(gen.UniqueIDWithPulse(pulse.OfNow()))
	retryUnorderedTable := object.NewRequestTable()
	oneRandomOrderedTable.GetList(contract.CallTolerable).Add(retryUnorderedRequestRef)

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
		PendingRequestTable           object.RequestTable
		expectedError                 bool
	}{
		{
			name:                        "OK tolerable",
			PendingRequestTable:         object.NewRequestTable(),
			requestRef:                  reference.NewSelf(gen.UniqueIDWithPulse(pulse.OfNow())),
			OrderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveOrderedPendingCount:   1,
			callFlags:                   payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			expectedResponse: &payload.VDelegatedCallResponse{
				DelegationSpec: payload.CallDelegationToken{
					TokenTypeAndFlags: payload.DelegationTokenTypeCall,
					PulseNumber:       pulse.OfNow(),
					ApproverSignature: deadBeef[:],
				},
			},
		},
		{
			name:                          "OK intolerable",
			PendingRequestTable:           object.NewRequestTable(),
			requestRef:                    reference.NewSelf(gen.UniqueIDWithPulse(pulse.OfNow())),
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
			expectedResponse: &payload.VDelegatedCallResponse{
				DelegationSpec: payload.CallDelegationToken{
					TokenTypeAndFlags: payload.DelegationTokenTypeCall,
					PulseNumber:       pulse.OfNow(),
					ApproverSignature: deadBeef[:],
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
				DelegationSpec: payload.CallDelegationToken{
					TokenTypeAndFlags: payload.DelegationTokenTypeCall,
					PulseNumber:       pulse.OfNow(),
					ApproverSignature: deadBeef[:],
				},
			},
		},
		{
			name:                          "retry intolerable",
			PendingRequestTable:           retryUnorderedTable,
			requestRef:                    retryUnorderedRequestRef,
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
			expectedResponse: &payload.VDelegatedCallResponse{
				DelegationSpec: payload.CallDelegationToken{
					TokenTypeAndFlags: payload.DelegationTokenTypeCall,
					PulseNumber:       pulse.OfNow(),
					ApproverSignature: deadBeef[:],
				},
			},
		},
		{
			name:                          "unexpected intolerable",
			PendingRequestTable:           object.NewRequestTable(),
			requestRef:                    reference.NewSelf(gen.UniqueIDWithPulse(pulse.OfNow() - 110)),
			UnorderedPendingEarliestPulse: pulse.Unknown,
			ActiveUnorderedPendingCount:   0,
			callFlags:                     payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
			expectedError:                 true,
		},
		{
			name:                          "unexpected tolerable",
			testRailCase:                  "C4985",
			PendingRequestTable:           object.NewRequestTable(),
			requestRef:                    reference.NewSelf(gen.UniqueIDWithPulse(pulse.OfNow() - 110)),
			UnorderedPendingEarliestPulse: pulse.Unknown,
			ActiveUnorderedPendingCount:   0,
			callFlags:                     payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			expectedError:                 true,
		},
		{
			name:                          "too old intolerable",
			testRailCase:                  "C4984",
			PendingRequestTable:           object.NewRequestTable(),
			requestRef:                    reference.NewSelf(gen.UniqueIDWithPulse(pulse.OfNow() - 110)),
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
			expectedError:                 true,
		},
		{
			name:                          "too old tolerable",
			PendingRequestTable:           object.NewRequestTable(),
			requestRef:                    reference.NewSelf(gen.UniqueIDWithPulse(pulse.OfNow() - 110)),
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			expectedError:                 true,
		},
		{
			name:                          "full table intolerable",
			PendingRequestTable:           oneRandomUnorderedTable,
			requestRef:                    reference.NewSelf(gen.UniqueIDWithPulse(pulse.OfNow() - 110)),
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
			expectedError:                 true,
		},
		{
			name:                          "full table tolerable",
			testRailCase:                  "C4986",
			PendingRequestTable:           oneRandomOrderedTable,
			requestRef:                    reference.NewSelf(gen.UniqueIDWithPulse(pulse.OfNow() - 110)),
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
			expectedError:                 true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.testRailCase != "" {
				t.Log(tc.testRailCase)
			}
			var (
				mc  = minimock.NewController(t)
				ctx = inslogger.TestContext(t)

				nodeRef = gen.UniqueReference()

				caller           = gen.UniqueReference()
				outgoing         = gen.UniqueID()
				objectRef        = reference.NewSelf(outgoing)
				orderedBargeIn   = smachine.BargeIn{}
				unorderedBargeIn = smachine.BargeIn{}
				sharedState      = &object.SharedState{
					Info: object.Info{
						PendingTable:                       tc.PendingRequestTable,
						OrderedPendingEarliestPulse:        tc.OrderedPendingEarliestPulse,
						UnorderedPendingEarliestPulse:      tc.UnorderedPendingEarliestPulse,
						ActiveOrderedPendingCount:          tc.ActiveOrderedPendingCount,
						ActiveUnorderedPendingCount:        tc.ActiveUnorderedPendingCount,
						KnownRequests:                      object.NewRequestTable(),
						ReadyToWork:                        smsync.NewConditional(1, "ReadyToWork").SyncLink(),
						OrderedExecute:                     smsync.NewConditional(1, "MutableExecution").SyncLink(),
						OrderedPendingListFilledCallback:   orderedBargeIn,
						UnorderedPendingListFilledCallback: unorderedBargeIn,
					},
				}
				callFlags = tc.callFlags
			)

			slotMachine := slotdebugger.New(ctx, t, tc.expectedError)
			if tc.expectedError {
				slotMachine.InitEmptyMessageSender(mc)
			} else {
				slotMachine.PrepareMockedMessageSender(mc)
			}

			expectedResponse := &payload.VDelegatedCallResponse{
				DelegationSpec: payload.CallDelegationToken{
					TokenTypeAndFlags: payload.DelegationTokenTypeCall,
					Approver:          nodeRef,
					DelegateTo:        caller,
					PulseNumber:       pulse.OfNow(),
					Callee:            objectRef,
					Caller:            caller,
					ApproverSignature: []byte{0xde, 0xad, 0xbe, 0xef},
				},
			}

			var jetMock jet.AffinityHelper = jet.NewAffinityHelperMock(mc).MeMock.Return(nodeRef)

			slotMachine.AddInterfaceDependency(&jetMock)

			smDelegatedCallRequest := SMVDelegatedCallRequest{
				Payload: &payload.VDelegatedCallRequest{
					RequestReference: tc.requestRef,
					CallFlags:        callFlags,
					Callee:           objectRef,
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

			slotMachine.MessageSender.SendTarget.Set(func(_ context.Context, msg payload.Marshaler, target reference.Global, _ ...messagesender.SendOption) error {
				res := msg.(*payload.VDelegatedCallResponse)
				// ensure that both times request is the same
				assert.Equal(t, caller, target)
				assert.Equal(t, expectedResponse, res)
				return nil
			})

			smWrapper := slotMachine.AddStateMachine(ctx, &smDelegatedCallRequest)

			slotMachine.RunTil(smWrapper.AfterStep(smDelegatedCallRequest.stepBuildResponse))

			assert.True(t, sharedState.PendingTable.GetList(callFlags.GetInterference()).Exist(tc.requestRef))

			require.NoError(t, catalogWrapper.CheckDone())
			mc.Finish()
		})
	}
}
