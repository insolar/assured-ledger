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
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/light/proc"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/object"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
)

func TestHasPendings_Proceed(t *testing.T) {
	ctx := flow.TestContextWithPulse(inslogger.TestContext(t), pulse.MinTimePulse+10)
	mc := minimock.NewController(t)

	var (
		index  *object.IndexAccessorMock
		sender *bus.SenderMock
	)

	setup := func() {
		index = object.NewIndexAccessorMock(mc)
		sender = bus.NewSenderMock(mc)
	}

	t.Run("ok, has pendings", func(t *testing.T) {
		setup()
		defer mc.Finish()

		pulseNumber := insolar.NewID(pulse.MinTimePulse, []byte{1}).Pulse()

		index.ForIDMock.Return(
			record.Index{
				Lifeline: record.Lifeline{
					EarliestOpenRequest: &pulseNumber,
				},
			},
			nil,
		)

		expectedMsg, _ := payload.NewMessage(&payload.PendingsInfo{
			HasPendings: true,
		})

		sender.ReplyMock.Inspect(func(ctx context.Context, origin payload.Meta, reply *message.Message) {
			assert.Equal(t, expectedMsg.Payload, reply.Payload)
		}).Return()

		p := proc.NewHasPendings(payload.Meta{}, gen.ID())
		p.Dep(index, sender)

		err := p.Proceed(ctx)
		assert.NoError(t, err)
	})

	t.Run("ok, no pendings", func(t *testing.T) {
		setup()
		defer mc.Finish()

		pulseNumber := insolar.NewID(pulse.MinTimePulse+100, []byte{1}).Pulse()

		index.ForIDMock.Return(
			record.Index{
				Lifeline: record.Lifeline{
					EarliestOpenRequest: &pulseNumber,
				},
			},
			nil,
		)

		expectedMsg, _ := payload.NewMessage(&payload.PendingsInfo{
			HasPendings: false,
		})

		sender.ReplyMock.Inspect(func(ctx context.Context, origin payload.Meta, reply *message.Message) {
			assert.Equal(t, expectedMsg.Payload, reply.Payload)
		}).Return()

		p := proc.NewHasPendings(payload.Meta{}, gen.ID())
		p.Dep(index, sender)

		err := p.Proceed(ctx)
		assert.NoError(t, err)
	})
}
