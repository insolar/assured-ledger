// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handlers

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	messageSender "github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/predicate"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/slotdebugger"
)

func TestVFindCallRequestAwaitCallSummarySM(t *testing.T) {
	var (
		ctx       = instestlogger.TestContext(t)
		mc        = minimock.NewController(t)
		objectRef = gen.UniqueGlobalRef()
	)

	slotMachine := slotdebugger.New(ctx, t)
	slotMachine.PrepareMockedMessageSender(mc)
	slotMachine.PrepareMockedRunner(ctx, mc)

	sendTarget := func(ctx context.Context, msg payload.Marshaler, target reference.Global, opts ...messageSender.SendOption) (err error) {
		return nil
	}

	sendRole := func(ctx context.Context, msg payload.Marshaler, role node.DynamicRole, object reference.Global, pn pulse.Number, opts ...messageSender.SendOption) (err error) {
		return nil
	}

	slotMachine.MessageSender.Mock().SendTargetMock.Set(sendTarget)
	slotMachine.MessageSender.Mock().SendRoleMock.Set(sendRole)

	smObject := object.NewStateMachineObject(objectRef)
	smObject.SetState(object.HasState)
	smObject.SetDescriptor(descriptor.NewObject(reference.Global{}, reference.Local{}, gen.UniqueGlobalRef(), []byte("213"), reference.Global{}))

	slotMachine.Start()
	defer slotMachine.Stop()

	pulseBeforeMigration := slotMachine.PulseSlot.PulseNumber()
	vFindCallRequest := payload.VFindCallRequest{
		LookAt:   pulseBeforeMigration,
		Callee:   objectRef,
		Outgoing: gen.UniqueGlobalRef(),
	}

	smVFindCallRequest := &SMVFindCallRequest{
		Meta: &payload.Meta{
			Sender: gen.UniqueGlobalRef(),
		},
		Payload: &vFindCallRequest,
	}

	smFindCallRequestSWrapper := slotMachine.AddStateMachine(ctx, smVFindCallRequest)
	smObjectWrapper := slotMachine.AddStateMachine(ctx, smObject)

	slotMachine.Migrate()

	slotMachine.RunTil(predicate.ChainOf(
		predicate.NewSMTypeFilter(&object.SMObject{}, smObjectWrapper.AfterAnyMigrate()),
		predicate.NewSMTypeFilter(&SMVFindCallRequest{}, smFindCallRequestSWrapper.BeforeStep(smVFindCallRequest.stepWaitCallResult)),
		predicate.NewSMTypeFilter(&object.SMObject{}, smObjectWrapper.AfterAnyStep()),
		predicate.NewSMTypeFilter(&SMVFindCallRequest{}, smFindCallRequestSWrapper.BeforeStep(smVFindCallRequest.stepGetRequestData)),
	))
}
