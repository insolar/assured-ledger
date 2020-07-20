// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package handlers

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/insolar/assured-ledger/ledger-core/conveyor"
	"github.com/insolar/assured-ledger/ledger-core/insolar/node"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/instrumentation/inslogger/instestlogger"
	messageSender "github.com/insolar/assured-ledger/ledger-core/network/messagesender"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/descriptor"
	"github.com/insolar/assured-ledger/ledger-core/virtual/object"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils/slotdebugger"
)

func TestVFindCallRequest_AwaitCallSummarySM(t *testing.T) {
	defer commontestutils.LeakTester(t)

	t.Log("C5435")

	var (
		ctx       = instestlogger.TestContext(t)
		mc        = minimock.NewController(t)
		objectRef = gen.UniqueGlobalRef()
		limiter   = conveyor.NewParallelProcessingLimiter(4)
	)

	slotMachine := slotdebugger.New(ctx, t)
	slotMachine.PrepareMockedMessageSender(mc)
	slotMachine.AddDependency(limiter)

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

	slotMachine.RunTil(smObjectWrapper.AfterAnyMigrate())
	slotMachine.RunTil(smFindCallRequestSWrapper.BeforeStep(smVFindCallRequest.stepWaitCallResult))
	slotMachine.RunTil(smObjectWrapper.AfterAnyStep())
	slotMachine.RunTil(smFindCallRequestSWrapper.BeforeStep(smVFindCallRequest.stepGetRequestData))
}
