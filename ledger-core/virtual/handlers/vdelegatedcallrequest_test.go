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
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
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

	retryOrderedRequestRef := gen.UniqueGlobalRefWithPulse(pulse.OfNow())
	retryOrderedTable := callregistry.NewRequestTable()
	oneRandomOrderedTable.GetList(isolation.CallTolerable).Add(retryOrderedRequestRef)

	retryUnorderedRequestRef := gen.UniqueGlobalRefWithPulse(pulse.OfNow())
	retryUnorderedTable := callregistry.NewRequestTable()
	oneRandomOrderedTable.GetList(isolation.CallTolerable).Add(retryUnorderedRequestRef)

	oneRandomOrderedRequest := gen.UniqueGlobalRefWithPulse(pulse.OfNow())
	oneRandomUnorderedRequest := gen.UniqueGlobalRefWithPulse(pulse.OfNow())

	for _, tc := range []struct {
		name                          string
		requestRef                    reference.Global
		OrderedPendingEarliestPulse   pulse.Number
		UnorderedPendingEarliestPulse pulse.Number
		ActiveOrderedPendingCount     uint8
		ActiveUnorderedPendingCount   uint8
		callFlags                     rms.CallFlags
		expectedResponse              *rms.VDelegatedCallResponse
		PendingRequestTable           callregistry.PendingTable
		expectedError                 bool
	}{
		{
			name:                        "OK tolerable",
			PendingRequestTable:         callregistry.NewRequestTable(),
			requestRef:                  oneRandomOrderedRequest,
			OrderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveOrderedPendingCount:   1,
			callFlags:                   rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
			expectedResponse: &rms.VDelegatedCallResponse{
				ResponseDelegationSpec: rms.CallDelegationToken{
					TokenTypeAndFlags: rms.DelegationTokenTypeCall,
					ApproverSignature: rms.NewBytes(deadBeef[:]),
					Outgoing:          rms.NewReference(oneRandomOrderedRequest),
				},
			},
		},
		{
			name:                          "OK intolerable",
			PendingRequestTable:           callregistry.NewRequestTable(),
			requestRef:                    oneRandomUnorderedRequest,
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
			expectedResponse: &rms.VDelegatedCallResponse{
				ResponseDelegationSpec: rms.CallDelegationToken{
					TokenTypeAndFlags: rms.DelegationTokenTypeCall,
					ApproverSignature: rms.NewBytes(deadBeef[:]),
					Outgoing:          rms.NewReference(oneRandomUnorderedRequest),
				},
			},
		},
		{
			name:                        "retry tolerable",
			PendingRequestTable:         retryOrderedTable,
			requestRef:                  retryOrderedRequestRef,
			OrderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveOrderedPendingCount:   1,
			callFlags:                   rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
			expectedResponse: &rms.VDelegatedCallResponse{
				ResponseDelegationSpec: rms.CallDelegationToken{
					TokenTypeAndFlags: rms.DelegationTokenTypeCall,
					ApproverSignature: rms.NewBytes(deadBeef[:]),
					Outgoing:          rms.NewReference(retryOrderedRequestRef),
				},
			},
		},
		{
			name:                          "retry intolerable",
			PendingRequestTable:           retryUnorderedTable,
			requestRef:                    retryUnorderedRequestRef,
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
			expectedResponse: &rms.VDelegatedCallResponse{
				ResponseDelegationSpec: rms.CallDelegationToken{
					TokenTypeAndFlags: rms.DelegationTokenTypeCall,
					PulseNumber:       pulse.OfNow(),
					ApproverSignature: rms.NewBytes(deadBeef[:]),
					Outgoing:          rms.NewReference(retryUnorderedRequestRef),
				},
			},
		},
		{
			name:                          "unexpected intolerable",
			PendingRequestTable:           callregistry.NewRequestTable(),
			requestRef:                    gen.UniqueGlobalRefWithPulse(pulse.OfNow() - 110),
			UnorderedPendingEarliestPulse: pulse.Unknown,
			ActiveUnorderedPendingCount:   0,
			callFlags:                     rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
			expectedError:                 true,
		},
		{
			name:                        "unexpected tolerable",
			PendingRequestTable:         callregistry.NewRequestTable(),
			requestRef:                  gen.UniqueGlobalRefWithPulse(pulse.OfNow() - 110),
			OrderedPendingEarliestPulse: pulse.Unknown,
			ActiveOrderedPendingCount:   0,
			callFlags:                   rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
			expectedError:               true,
		},
		{
			name:                          "too old intolerable",
			PendingRequestTable:           callregistry.NewRequestTable(),
			requestRef:                    gen.UniqueGlobalRefWithPulse(pulse.OfNow() - 110),
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
			expectedError:                 true,
		},
		{
			name:                        "too old tolerable",
			PendingRequestTable:         callregistry.NewRequestTable(),
			requestRef:                  gen.UniqueGlobalRefWithPulse(pulse.OfNow() - 110),
			OrderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveOrderedPendingCount:   1,
			callFlags:                   rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
			expectedError:               true,
		},
		{
			name:                          "full table intolerable",
			PendingRequestTable:           oneRandomUnorderedTable,
			requestRef:                    gen.UniqueGlobalRefWithPulse(pulse.OfNow() - 110),
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
			expectedError:                 true,
		},
		{
			name:                        "full table tolerable",
			PendingRequestTable:         oneRandomOrderedTable,
			requestRef:                  gen.UniqueGlobalRefWithPulse(pulse.OfNow() - 110),
			OrderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveOrderedPendingCount:   1,
			callFlags:                   rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
			expectedError:               true,
		},
		{
			name:                          "wrong call interference flag_expected intolerable, get tolerable",
			PendingRequestTable:           callregistry.NewRequestTable(),
			requestRef:                    gen.UniqueGlobalRefWithPulse(pulse.OfNow()),
			UnorderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveUnorderedPendingCount:   1,
			callFlags:                     rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
			expectedError:                 true,
		},
		{
			name:                        "wrong call interference flag_expected tolerable, get intolerable",
			PendingRequestTable:         callregistry.NewRequestTable(),
			requestRef:                  gen.UniqueGlobalRefWithPulse(pulse.OfNow()),
			OrderedPendingEarliestPulse: pulse.OfNow() - 100,
			ActiveOrderedPendingCount:   1,
			callFlags:                   rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
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
				Payload: &rms.VDelegatedCallRequest{
					CallOutgoing: rms.NewReference(tc.requestRef),
					CallFlags:    callFlags,
					Callee:       rms.NewReference(objectRef),
				},
				Meta: &rms.Meta{
					Sender: rms.NewReference(caller),
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
			expectedResponse.ResponseDelegationSpec.Approver.Set(nodeRef)
			expectedResponse.ResponseDelegationSpec.DelegateTo.Set(caller)
			expectedResponse.ResponseDelegationSpec.Caller.Set(caller)
			expectedResponse.ResponseDelegationSpec.PulseNumber = pulse.OfNow()
			expectedResponse.ResponseDelegationSpec.Callee.Set(objectRef)
			expectedResponse.Callee.Set(objectRef)

			slotMachine.MessageSender.SendTarget.Set(func(_ context.Context, msg rmsreg.GoGoSerializable, target reference.Global, _ ...messagesender.SendOption) error {
				res := msg.(*rms.VDelegatedCallResponse)
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
