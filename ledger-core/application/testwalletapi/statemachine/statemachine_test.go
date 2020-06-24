// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package statemachine

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	"github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/slotdebugger"
)

func TestSMTestAPICall_Migrate(t *testing.T) {
	var (
		mc  = minimock.NewController(t)
		ctx = instestlogger.TestContext(t)
	)

	slotMachine := slotdebugger.NewWithIgnoreAllErrors(ctx, t)

	request := payload.VCallRequest{
		CallType:            payload.CTMethod,
		Callee:              gen.UniqueGlobalRef(),
		Caller:              APICaller,
		CallFlags:           payload.BuildCallFlags(contract.CallTolerable, contract.CallDirty),
		CallSiteDeclaration: testwallet.GetClass(),
		CallSiteMethod:      "New",
		CallOutgoing:        gen.UniqueLocalRef(),
		Arguments:           insolar.MustSerialize([]interface{}{}),
	}

	slotMachine.PrepareMockedMessageSender(mc)

	slotMachine.Start()
	defer slotMachine.Stop()

	slotMachine.MessageSender.SendRole.Set(func(_ context.Context, msg payload.Marshaler, role node.DynamicRole, object reference.Global, pn pulse.Number, _ ...messagesender.SendOption) error {
		res := msg.(*payload.VCallRequest)
		assert.Equal(t, request.Callee, res.Callee)
		assert.Equal(t, node.DynamicRoleVirtualExecutor, role)
		assert.Equal(t, request.Callee, object)
		return nil
	})

	smRequest := SMTestAPICall{
		requestPayload: request,
	}

	smWrapper := slotMachine.AddStateMachine(ctx, &smRequest)

	slotMachine.RunTil(smWrapper.BeforeStep(smRequest.stepSendRequest))

	outgoingCall := smRequest.requestPayload.CallOutgoing

	outgoingRef := reference.NewRecordOf(APICaller, outgoingCall)
	_, bargeIn := slotMachine.SlotMachine.GetPublishedGlobalAliasAndBargeIn(outgoingRef)
	assert.NotNil(t, bargeIn)

	slotMachine.MessageSender.SendRole.Set(func(_ context.Context, msg payload.Marshaler, role node.DynamicRole, object reference.Global, pn pulse.Number, _ ...messagesender.SendOption) error {
		res := msg.(*payload.VCallRequest)
		// ensure that both times request is the same
		assert.Equal(t, outgoingCall, res.CallOutgoing)
		assert.Equal(t, node.DynamicRoleVirtualExecutor, role)
		assert.Equal(t, request.Callee, object)
		return nil
	})

	slotMachine.RunTil(smWrapper.AfterStep(smRequest.stepSendRequest))

	slotMachine.Migrate()

	slotMachine.RunTil(smWrapper.AfterMigrate(smRequest.migrationDefault))

	slotMachine.RunTil(smWrapper.BeforeStep(smRequest.stepProcessResult))
}
