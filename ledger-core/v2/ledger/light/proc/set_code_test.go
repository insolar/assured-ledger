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
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/object"
	"github.com/insolar/assured-ledger/ledger-core/v2/testutils"
)

func TestSetCode_Proceed(t *testing.T) {
	ctx := flow.TestContextWithPulse(inslogger.TestContext(t), gen.PulseNumber())
	mc := minimock.NewController(t)

	var (
		writer  *executor.WriteAccessorMock
		records *object.AtomicRecordModifierMock
		pcs     insolar.PlatformCryptographyScheme
		sender  *bus.SenderMock
	)

	setup := func() {
		writer = executor.NewWriteAccessorMock(mc)
		records = object.NewAtomicRecordModifierMock(mc)
		pcs = testutils.NewPlatformCryptographyScheme()
		sender = bus.NewSenderMock(mc)
	}

	t.Run("Simple success", func(t *testing.T) {
		invoked := false
		setup()
		defer mc.Finish()

		msg := payload.Meta{
			Receiver: gen.Reference(),
		}

		jetID := gen.JetID()
		recVirtual := record.Wrap(&record.Code{})
		recordID := gen.ID()
		rec := record.Material{
			Virtual: recVirtual,
			ID:      recordID,
			JetID:   jetID,
		}

		writer.BeginMock.Return(func() {
			invoked = true
		}, nil)

		records.SetAtomicMock.Inspect(func(ctx context.Context, records ...record.Material) {
			assert.Equal(t, rec, records[0])
		}).Return(nil)

		expectedMsg, _ := payload.NewMessage(&payload.ID{
			ID: recordID,
		})

		sender.ReplyMock.Inspect(func(ctx context.Context, origin payload.Meta, reply *message.Message) {
			assert.Equal(t, expectedMsg.Payload, reply.Payload)
			assert.Equal(t, msg, origin)
		}).Return()

		p := proc.NewSetCode(msg, recVirtual, recordID, jetID)
		p.Dep(writer, records, pcs, sender)

		err := p.Proceed(ctx)
		assert.NoError(t, err)
		assert.True(t, invoked, "must be invoked")
	})

	t.Run("Error flow cancelled", func(t *testing.T) {
		setup()
		defer mc.Finish()

		msg := payload.Meta{
			Receiver: gen.Reference(),
		}

		jetID := gen.JetID()
		recVirtual := record.Wrap(&record.Code{})
		recordID := gen.ID()

		writer.BeginMock.Return(func() {}, executor.ErrWriteClosed)

		p := proc.NewSetCode(msg, recVirtual, recordID, jetID)
		p.Dep(writer, records, pcs, sender)

		err := p.Proceed(ctx)
		assert.Error(t, err)
	})
}
