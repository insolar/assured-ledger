// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func makeVStateRequestEvent(pulseNumber pulse.Number, ref reference.Global, flags payload.StateRequestContentFlags) *message.Message {
	payload := &payload.VStateRequest{
		AsOf:             pulseNumber,
		Callee:           ref,
		RequestedContent: flags,
	}

	return utils.NewRequestWrapper(pulseNumber, payload).Finalize()
}

func TestVirtual_VStateRequest_WithoutBody(t *testing.T) {
	t.Log("C4861")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	reportChan := make(chan *payload.VStateReport, 0)

	server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
		for _, msg := range messages {
			pl, err := payload.UnmarshalFromMeta(msg.Payload)
			require.NoError(t, err)

			switch plData := pl.(type) {
			case *payload.VStateReport:
				reportChan <- plData
				continue
			}
			server.SendMessage(ctx, msg)
		}
		return nil
	})

	testBalance := uint32(555)
	rawWalletState := makeRawWalletState(t, testBalance)
	objectRef := gen.UniqueReference()
	stateID := gen.UniqueIDWithPulse(server.GetPulse().PulseNumber)
	{
		// send VStateReport: save wallet
		msg := makeVStateReportEvent(server.GetPulse().PulseNumber, objectRef, stateID, rawWalletState, server.JetCoordinatorMock.Me())
		server.SendMessage(ctx, msg)
	}

	msg := makeVStateRequestEvent(server.GetPulse().PulseNumber, objectRef, 0)
	server.SendMessage(ctx, msg)

	select {
	case data := <-reportChan:
		assert.Equal(t, &payload.VStateReport{
			Status:           payload.Ready,
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

	server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
		for _, msg := range messages {
			pl, err := payload.UnmarshalFromMeta(msg.Payload)
			require.NoError(t, err)

			switch plData := pl.(type) {
			case *payload.VStateReport:
				reportChan <- plData
				continue
			}
			server.SendMessage(ctx, msg)
		}
		return nil
	})

	testBalance := uint32(555)
	rawWalletState := makeRawWalletState(t, testBalance)
	objectRef := gen.UniqueReference()
	stateID := gen.UniqueIDWithPulse(server.GetPulse().PulseNumber)
	{
		// send VStateReport: save wallet
		msg := makeVStateReportEvent(server.GetPulse().PulseNumber, objectRef, stateID, rawWalletState, server.JetCoordinatorMock.Me())
		server.SendMessage(ctx, msg)
	}

	msg := makeVStateRequestEvent(server.GetPulse().PulseNumber, objectRef, payload.RequestLatestDirtyState)
	server.SendMessage(ctx, msg)

	select {
	case data := <-reportChan:
		assert.Equal(t, &payload.VStateReport{
			Status:           payload.Ready,
			AsOf:             server.GetPulse().PulseNumber,
			Callee:           objectRef,
			LatestDirtyState: objectRef,
			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState: &payload.ObjectState{
					Reference: stateID,
					State:     rawWalletState,
					Class:     testwallet.ClassReference,
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

	reportChan := make(chan *payload.VStateReport, 0)

	server.PublisherMock.SetChecker(func(topic string, messages ...*message.Message) error {
		for _, msg := range messages {
			pl, err := payload.UnmarshalFromMeta(msg.Payload)
			require.NoError(t, err)

			switch plData := pl.(type) {
			case *payload.VStateReport:
				reportChan <- plData
				continue
			}
			server.SendMessage(ctx, msg)
		}
		return nil
	})

	objectRef := gen.UniqueReference()

	msg := makeVStateRequestEvent(server.GetPulse().PulseNumber, objectRef, payload.RequestLatestDirtyState)
	server.SendMessage(ctx, msg)

	select {
	case data := <-reportChan:
		assert.Equal(t, &payload.VStateReport{
			Status: payload.Missing,
			AsOf:   server.GetPulse().PulseNumber,
			Callee: objectRef,
		}, data)
	case <-time.After(10 * time.Second):
		require.Failf(t, "", "timeout")
	}
}
