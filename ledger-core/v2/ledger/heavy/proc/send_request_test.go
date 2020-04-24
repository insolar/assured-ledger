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
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/bus"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/record"
	"github.com/insolar/assured-ledger/ledger-core/v2/instrumentation/inslogger"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/heavy/proc"
	"github.com/insolar/assured-ledger/ledger-core/v2/ledger/object"
)

func TestGetRequest_Proceed(t *testing.T) {
	mc := minimock.NewController(t)
	ctx := inslogger.TestContext(t)

	var (
		sender  *bus.SenderMock
		records *object.RecordAccessorMock
	)

	resetComponents := func() {
		sender = bus.NewSenderMock(mc)
		records = object.NewRecordAccessorMock(t)
	}

	newProc := func(msg payload.Meta) *proc.SendRequest {
		p := proc.NewSendRequest(msg)
		p.Dep(records, sender)
		return p
	}

	resetComponents()
	t.Run("request does not exist", func(t *testing.T) {
		sender.ReplyMock.Set(func(_ context.Context, _ payload.Meta, msg *message.Message) {
			rep := payload.Error{}
			err := rep.Unmarshal(msg.Payload)
			require.NoError(t, err)
			require.Equal(t, rep.Code, payload.CodeNotFound)
			require.Equal(t, rep.Text, object.ErrNotFound.Error())
		})
		p := newProc(payload.Meta{})
		records.ForIDMock.Return(record.Material{}, object.ErrNotFound)

		err := p.Proceed(ctx)
		require.NoError(t, err)

		mc.Finish()
	})

	resetComponents()
	t.Run("happy basic", func(t *testing.T) {
		reqID := gen.ID()
		msg := payload.GetRequest{
			RequestID: reqID,
		}
		buf, err := msg.Marshal()
		require.NoError(t, err)
		receivedMeta := payload.Meta{Payload: buf}
		p := newProc(receivedMeta)

		ref := gen.Reference()
		req := record.Virtual{
			Union: &record.Virtual_IncomingRequest{
				IncomingRequest: &record.IncomingRequest{
					Object: &ref,
				},
			},
		}

		records.ForIDMock.Return(record.Material{
			Virtual: req,
		}, nil)
		sender.ReplyMock.Set(func(_ context.Context, origin payload.Meta, rep *message.Message) {
			require.Equal(t, receivedMeta, origin)

			resp, err := payload.Unmarshal(rep.Payload)
			require.NoError(t, err)

			res, ok := resp.(*payload.Request)
			require.True(t, ok)
			require.Equal(t, msg.RequestID, res.RequestID)
			require.Equal(t, req, res.Request)
		})

		err = p.Proceed(ctx)
		require.NoError(t, err)

		mc.Finish()
	})
}
