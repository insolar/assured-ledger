// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/gen"
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/v2/virtual/statemachine"
)

func makeVStateRequestEvent(t *testing.T, pn pulse.Number, ref reference.Global, flags payload.StateRequestContentFlags) *statemachine.DispatcherMessage {
	payLoadMeta := &payload.VStateRequest{
		AsOf:             pn,
		Callee:           ref,
		RequestedContent: flags,
	}

	rawPayLoad, err := payload.Marshal(payLoadMeta)
	require.NoError(t, err)

	return &statemachine.DispatcherMessage{
		MessageMeta: message.Metadata{},
		PayloadMeta: &payload.Meta{
			Polymorph: uint32(payload.TypeMeta),
			Payload:   rawPayLoad},
	}
}

func TestVirtual_VStateRequest_WithoutBody(t *testing.T) {
	t.Log("C4861")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	reportChan := make(chan *payload.VStateReport, 0)

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		for _, msg := range messages {
			pl, err := payload.UnmarshalFromMeta(msg.Payload)
			require.NoError(t, err)

			switch plData := pl.(type) {
			case *payload.VStateReport:
				reportChan <- plData
				continue
			}
			server.SendMessage(context.Background(), msg)
		}
		return nil
	}

	testBalance := uint32(555)
	rawWalletState := makeRawWalletState(t, testBalance)
	objectRef := gen.Reference()
	stateID := gen.UniqueIDWithPulse(server.GetPulse().PulseNumber)
	{
		// send VStateReport: save wallet
		msg := makeVStateReportEvent(t, objectRef, stateID, rawWalletState)
		require.NoError(t, server.AddInput(ctx, msg))
	}

	msg := makeVStateRequestEvent(t, server.GetPulse().PulseNumber, objectRef, 0)

	require.NoError(t, server.AddInput(ctx, msg))

	select {
	case data := <-reportChan:
		assert.Equal(t, &payload.VStateReport{
			Polymorph:        uint32(payload.TypeVStateReport),
			AsOf:             server.GetPulse().PulseNumber,
			Callee:           objectRef,
			LatestDirtyState: objectRef,
		}, data)
	case <-time.After(10 * time.Second):
		require.Failf(t, "", "timeout")
	}

}

func TestVirtual_VStateRequest_WithBody(t *testing.T) {
	t.Log("C4862")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	reportChan := make(chan *payload.VStateReport, 0)

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		for _, msg := range messages {
			pl, err := payload.UnmarshalFromMeta(msg.Payload)
			require.NoError(t, err)

			switch plData := pl.(type) {
			case *payload.VStateReport:
				reportChan <- plData
				continue
			}
			server.SendMessage(context.Background(), msg)
		}
		return nil
	}

	testBalance := uint32(555)
	rawWalletState := makeRawWalletState(t, testBalance)
	objectRef := gen.Reference()
	stateID := gen.UniqueIDWithPulse(server.GetPulse().PulseNumber)
	{
		// send VStateReport: save wallet
		msg := makeVStateReportEvent(t, objectRef, stateID, rawWalletState)
		require.NoError(t, server.AddInput(ctx, msg))
	}

	msg := makeVStateRequestEvent(t, server.GetPulse().PulseNumber, objectRef, payload.RequestLatestDirtyState)

	require.NoError(t, server.AddInput(ctx, msg))

	select {
	case data := <-reportChan:
		assert.Equal(t, &payload.VStateReport{
			Polymorph:        uint32(payload.TypeVStateReport),
			AsOf:             server.GetPulse().PulseNumber,
			Callee:           objectRef,
			LatestDirtyState: objectRef,
			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState: &payload.ObjectState{
					Reference: stateID,
					State:     rawWalletState,
					Prototype: testwallet.PrototypeReference,
				},
			},
		}, data)
	case <-time.After(10 * time.Second):
		require.Failf(t, "", "timeout")
	}
}

func TestVirtual_VStateRequest_Unknown(t *testing.T) {
	t.Log("C4863")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	reportChan := make(chan *payload.VStateUnavailable, 0)

	server.PublisherMock.Checker = func(topic string, messages ...*message.Message) error {
		for _, msg := range messages {
			pl, err := payload.UnmarshalFromMeta(msg.Payload)
			require.NoError(t, err)

			switch plData := pl.(type) {
			case *payload.VStateUnavailable:
				reportChan <- plData
				continue
			}
			server.SendMessage(context.Background(), msg)
		}
		return nil
	}

	objectRef := gen.Reference()

	msg := makeVStateRequestEvent(t, server.GetPulse().PulseNumber, objectRef, payload.RequestLatestDirtyState)

	require.NoError(t, server.AddInput(ctx, msg))

	select {
	case data := <-reportChan:
		assert.Equal(t, &payload.VStateUnavailable{
			Polymorph: uint32(payload.TypeVStateUnavailable),
			Reason:    payload.Missing,
			Lifeline:  objectRef,
		}, data)
	case <-time.After(10 * time.Second):
		require.Failf(t, "", "timeout")
	}
}
