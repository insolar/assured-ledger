// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package proc_test

import (
	"context"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/flow"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/light/executor"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/light/proc"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

func TestGetPendings_Proceed(t *testing.T) {
	ctx := flow.TestContextWithPulse(inslogger.TestContext(t), pulse.MinTimePulse+10)
	mc := minimock.NewController(t)

	var (
		filaments *executor.FilamentCalculatorMock
		sender    *bus.SenderMock
	)

	setup := func() {
		filaments = executor.NewFilamentCalculatorMock(mc)
		sender = bus.NewSenderMock(mc)
	}

	emptyRefs := make([]insolar.ID, 0)
	t.Run("ok, pendings is empty", func(t *testing.T) {
		setup()
		defer mc.Finish()

		filaments.OpenedRequestsMock.Return([]record.CompositeFilamentRecord{}, nil)

		expectedMsg, _ := payload.NewMessage(&payload.Error{
			Text: insolar.ErrNoPendingRequest.Error(),
			Code: payload.CodeNoPendings,
		})

		sender.ReplyMock.Inspect(func(ctx context.Context, origin payload.Meta, reply *message.Message) {
			assert.Equal(t, expectedMsg.Payload, reply.Payload)
		}).Return()

		p := proc.NewGetPendings(payload.Meta{}, gen.ID(), 1, emptyRefs)

		p.Dep(filaments, sender)

		err := p.Proceed(ctx)
		assert.NoError(t, err)
	})

	t.Run("ok, pendings found", func(t *testing.T) {
		setup()
		defer mc.Finish()
		pendings := []record.CompositeFilamentRecord{
			{RecordID: gen.ID()},
			{RecordID: gen.ID()},
			{RecordID: gen.ID()},
			{RecordID: gen.ID()},
		}

		ids := make([]insolar.ID, len(pendings))
		for i, pend := range pendings {
			ids[i] = pend.RecordID
		}

		expectedMsg, _ := payload.NewMessage(&payload.IDs{
			IDs: ids,
		})

		filaments.OpenedRequestsMock.Return(pendings, nil)

		sender.ReplyMock.Inspect(func(ctx context.Context, origin payload.Meta, reply *message.Message) {
			assert.Equal(t, expectedMsg.Payload, reply.Payload)
		}).Return()

		p := proc.NewGetPendings(payload.Meta{}, gen.ID(), 10, emptyRefs)
		p.Dep(filaments, sender)

		err := p.Proceed(ctx)
		assert.NoError(t, err)
	})

	t.Run("requested less than exists returns correct", func(t *testing.T) {
		setup()
		defer mc.Finish()
		pendings := []record.CompositeFilamentRecord{
			{RecordID: gen.ID()},
			{RecordID: gen.ID()},
			{RecordID: gen.ID()},
			{RecordID: gen.ID()},
		}

		ids := make([]insolar.ID, len(pendings))
		for i, pend := range pendings {
			ids[i] = pend.RecordID
		}

		expectedMsg, _ := payload.NewMessage(&payload.IDs{
			IDs: ids[:3],
		})

		filaments.OpenedRequestsMock.Return(pendings, nil)

		sender.ReplyMock.Inspect(func(ctx context.Context, origin payload.Meta, reply *message.Message) {
			assert.Equal(t, expectedMsg.Payload, reply.Payload)
		}).Return()

		p := proc.NewGetPendings(payload.Meta{}, gen.ID(), 3, emptyRefs)
		p.Dep(filaments, sender)

		err := p.Proceed(ctx)
		assert.NoError(t, err)
	})

	t.Run("skip few requests", func(t *testing.T) {
		setup()
		defer mc.Finish()

		Pending0 := gen.ID()
		Pending1 := gen.ID()
		Pending2 := gen.ID()
		Pending3 := gen.ID()

		pendings := []record.CompositeFilamentRecord{
			{RecordID: Pending0},
			{RecordID: Pending1},
			{RecordID: Pending2},
			{RecordID: Pending3},
		}

		ids := make([]insolar.ID, len(pendings))
		for i, pend := range pendings {
			ids[i] = pend.RecordID
		}

		expectedMsg, _ := payload.NewMessage(&payload.IDs{
			IDs: ids[2:4],
		})

		filaments.OpenedRequestsMock.Return(pendings, nil)

		sender.ReplyMock.Inspect(func(ctx context.Context, origin payload.Meta, reply *message.Message) {
			assert.Equal(t, expectedMsg.Payload, reply.Payload)
		}).Return()

		p := proc.NewGetPendings(payload.Meta{}, gen.ID(), 2, []insolar.ID{Pending0, Pending1})
		p.Dep(filaments, sender)

		err := p.Proceed(ctx)
		assert.NoError(t, err)
	})
}
