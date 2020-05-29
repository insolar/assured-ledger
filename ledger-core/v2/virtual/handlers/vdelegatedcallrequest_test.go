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

func TestSMVDelegatedCallRequest_OK(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)

		nodeRef = gen.UniqueReference()

		caller      = gen.UniqueReference()
		outgoing    = gen.UniqueID()
		requestRef  = reference.NewSelf(gen.UniqueIDWithPulse(pulse.OfNow()))
		objectRef   = reference.NewSelf(outgoing)
		bargeIn     = smachine.BargeIn{}
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:                     object.NewRequestTable(),
				OrderedPendingEarliestPulse:      pulse.OfNow() - 100,
				ActiveOrderedPendingCount:        1,
				KnownRequests:                    object.NewRequestTable(),
				ReadyToWork:                      smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute:                   smsync.NewConditional(1, "MutableExecution").SyncLink(),
				OrderedPendingListFilledCallback: bargeIn,
			},
		}
		callFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)
	)

	slotMachine := slotdebugger.New(ctx, t, false)
	slotMachine.PrepareMockedMessageSender(mc)

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
			RequestReference: requestRef,
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

	slotMachine.MessageSender.SendTarget.Set(func(_ context.Context, msg payload.Marshaler, target reference.Global, _ ...messagesender.SendOption) error {
		res := msg.(*payload.VDelegatedCallResponse)
		// ensure that both times request is the same
		assert.Equal(t, caller, target)
		assert.Equal(t, expectedResponse, res)
		return nil
	})

	smWrapper := slotMachine.AddStateMachine(ctx, &smDelegatedCallRequest)

	slotMachine.RunTil(smWrapper.AfterStep(smDelegatedCallRequest.stepBuildResponse))

	assert.True(t, sharedState.PendingTable.GetList(contract.CallTolerable).Exist(requestRef))

	require.NoError(t, catalogWrapper.CheckDone())
	mc.Finish()
}

func TestSMVDelegatedCallRequest_TooOld(t *testing.T) {
	t.Log("C4984")
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)

		nodeRef = gen.UniqueReference()

		caller      = gen.UniqueReference()
		outgoing    = gen.UniqueID()
		requestID   = gen.UniqueIDWithPulse(pulse.OfNow())
		requestRef  = reference.NewSelf(requestID)
		objectRef   = reference.NewSelf(outgoing)
		bargeIn     = smachine.BargeIn{}
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:                     object.NewRequestTable(),
				ActiveOrderedPendingCount:        1,
				OrderedPendingEarliestPulse:      pulse.OfNow() + 100,
				KnownRequests:                    object.NewRequestTable(),
				ReadyToWork:                      smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute:                   smsync.NewConditional(1, "MutableExecution").SyncLink(),
				OrderedPendingListFilledCallback: bargeIn,
			},
		}
		callFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)
	)

	slotMachine := slotdebugger.New(ctx, t, true)
	slotMachine.InitEmptyMessageSender(mc)

	var jetMock jet.AffinityHelper = jet.NewAffinityHelperMock(mc).MeMock.Return(nodeRef)

	slotMachine.AddInterfaceDependency(&jetMock)

	smDelegatedCallRequest := SMVDelegatedCallRequest{
		Payload: &payload.VDelegatedCallRequest{
			RequestReference: requestRef,
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

	smWrapper := slotMachine.AddStateMachine(ctx, &smDelegatedCallRequest)

	slotMachine.RunTil(smWrapper.AfterStop())
	mc.Finish()
}

func TestSMVDelegatedCallRequest_Unexpected(t *testing.T) {
	t.Log("C4985")
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)

		nodeRef = gen.UniqueReference()

		caller      = gen.UniqueReference()
		outgoing    = gen.UniqueID()
		requestRef  = gen.UniqueReference()
		objectRef   = reference.NewSelf(outgoing)
		bargeIn     = smachine.BargeIn{}
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:                     object.NewRequestTable(),
				ActiveOrderedPendingCount:        1,
				OrderedPendingEarliestPulse:      pulse.Unknown,
				KnownRequests:                    object.NewRequestTable(),
				ReadyToWork:                      smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute:                   smsync.NewConditional(1, "MutableExecution").SyncLink(),
				OrderedPendingListFilledCallback: bargeIn,
			},
		}
		callFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)
	)

	slotMachine := slotdebugger.New(ctx, t, true)
	slotMachine.InitEmptyMessageSender(mc)

	var jetMock jet.AffinityHelper = jet.NewAffinityHelperMock(mc).MeMock.Return(nodeRef)

	slotMachine.AddInterfaceDependency(&jetMock)

	smDelegatedCallRequest := SMVDelegatedCallRequest{
		Payload: &payload.VDelegatedCallRequest{
			RequestReference: requestRef,
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

	smWrapper := slotMachine.AddStateMachine(ctx, &smDelegatedCallRequest)

	slotMachine.RunTil(smWrapper.AfterStop())
	mc.Finish()
}

func TestSMVDelegatedCallRequest_FullTable(t *testing.T) {
	t.Log("C4986")
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)

		nodeRef = gen.UniqueReference()

		caller     = gen.UniqueReference()
		outgoing   = gen.UniqueID()
		requestRef = gen.UniqueReference()
		objectRef  = reference.NewSelf(outgoing)
		bargeIn    = smachine.BargeIn{}
		table      = object.NewRequestTable()

		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:                     table,
				ActiveOrderedPendingCount:        1,
				OrderedPendingEarliestPulse:      pulse.OfNow() - 100,
				KnownRequests:                    object.NewRequestTable(),
				ReadyToWork:                      smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute:                   smsync.NewConditional(1, "MutableExecution").SyncLink(),
				OrderedPendingListFilledCallback: bargeIn,
			},
		}
		callFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)
	)

	assert.True(t, sharedState.Info.PendingTable.GetList(contract.CallTolerable).Add(gen.UniqueReference()))

	slotMachine := slotdebugger.New(ctx, t, true)
	slotMachine.InitEmptyMessageSender(mc)

	var jetMock jet.AffinityHelper = jet.NewAffinityHelperMock(mc).MeMock.Return(nodeRef)

	slotMachine.AddInterfaceDependency(&jetMock)

	smDelegatedCallRequest := SMVDelegatedCallRequest{
		Payload: &payload.VDelegatedCallRequest{
			RequestReference: requestRef,
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

	smWrapper := slotMachine.AddStateMachine(ctx, &smDelegatedCallRequest)

	slotMachine.RunTil(smWrapper.AfterStop())
	mc.Finish()
}

func TestSMVDelegatedCallRequest_Retry(t *testing.T) {
	t.Log("C4987")
	var (
		mc  = minimock.NewController(t)
		ctx = inslogger.TestContext(t)

		nodeRef = gen.UniqueReference()

		caller      = gen.UniqueReference()
		outgoing    = gen.UniqueID()
		requestRef  = gen.UniqueReference()
		objectRef   = reference.NewSelf(outgoing)
		bargeIn     = smachine.BargeIn{}
		sharedState = &object.SharedState{
			Info: object.Info{
				PendingTable:                     object.NewRequestTable(),
				OrderedPendingEarliestPulse:      pulse.OfNow() - 100,
				ActiveOrderedPendingCount:        1,
				KnownRequests:                    object.NewRequestTable(),
				ReadyToWork:                      smsync.NewConditional(1, "ReadyToWork").SyncLink(),
				OrderedExecute:                   smsync.NewConditional(1, "MutableExecution").SyncLink(),
				OrderedPendingListFilledCallback: bargeIn,
			},
		}
		callFlags = payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty)
	)

	slotMachine := slotdebugger.New(ctx, t, false)
	slotMachine.PrepareMockedMessageSender(mc)

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
			RequestReference: requestRef,
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

	slotMachine.MessageSender.SendTarget.Set(func(_ context.Context, msg payload.Marshaler, target reference.Global, _ ...messagesender.SendOption) error {
		res := msg.(*payload.VDelegatedCallResponse)
		// ensure that both times request is the same
		assert.Equal(t, caller, target)
		assert.Equal(t, expectedResponse, res)
		return nil
	})

	smWrapper := slotMachine.AddStateMachine(ctx, &smDelegatedCallRequest)

	slotMachine.RunTil(smWrapper.AfterStep(smDelegatedCallRequest.stepBuildResponse))

	assert.True(t, sharedState.PendingTable.GetList(contract.CallTolerable).Exist(requestRef))

	smDelegatedCallRequestRetry := SMVDelegatedCallRequest{
		Payload: &payload.VDelegatedCallRequest{
			RequestReference: requestRef,
			CallFlags:        callFlags,
			Callee:           objectRef,
		},
		Meta: &payload.Meta{
			Sender: caller,
		},
	}

	smWrapperRetry := slotMachine.AddStateMachine(ctx, &smDelegatedCallRequestRetry)

	slotMachine.RunTil(smWrapperRetry.AfterStep(smDelegatedCallRequest.stepBuildResponse))

	assert.True(t, sharedState.PendingTable.GetList(contract.CallTolerable).Exist(requestRef))

	require.NoError(t, catalogWrapper.CheckDone())
	mc.Finish()
}
