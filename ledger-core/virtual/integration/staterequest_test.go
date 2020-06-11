// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
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

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	var (
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
		pn           = server.GetPulse().PulseNumber
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	countBefore := server.PublisherMock.GetCount()
	server.IncrementPulse(ctx)
	if !server.PublisherMock.WaitCount(countBefore+1, 10*time.Second) {
		t.Fatal("timeout waiting for VStateReport")
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
		assert.Equal(t, &payload.VStateReport{
			Status:           payload.Ready,
			AsOf:             server.GetPrevPulse().PulseNumber,
			Callee:           objectGlobal,
			LatestDirtyState: objectGlobal,
		}, report)

		return false
	})

	countBefore = server.PublisherMock.GetCount()
	msg := makeVStateRequestEvent(pn, objectGlobal, 0)
	server.SendMessage(ctx, msg)

	if !server.PublisherMock.WaitCount(countBefore+1, 10*time.Second) {
		t.Fatal("timeout waiting for VStateReport")
	}
}

func TestVirtual_VStateRequest_WithBody(t *testing.T) {
	t.Log("C4862")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	var (
		objectLocal    = server.RandomLocalWithPulse()
		objectGlobal   = reference.NewSelf(objectLocal)
		pn             = server.GetPulse().PulseNumber
		rawWalletState = makeRawWalletState(initialBalance)
	)

	Method_PrepareObject(ctx, server, payload.Ready, objectGlobal)

	countBefore := server.PublisherMock.GetCount()
	server.IncrementPulse(ctx)
	if !server.PublisherMock.WaitCount(countBefore+1, 10*time.Second) {
		t.Fatal("timeout waiting for VStateReport")
	}

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
		assert.Equal(t, &payload.VStateReport{
			Status:           payload.Ready,
			AsOf:             pn,
			Callee:           objectGlobal,
			LatestDirtyState: objectGlobal,
			ProvidedContent: &payload.VStateReport_ProvidedContentBody{
				LatestDirtyState: &payload.ObjectState{
					Reference: reference.Local{},
					State:     rawWalletState,
					Class:     testwallet.ClassReference,
				},
			},
		}, report)

		return false
	})

	countBefore = server.PublisherMock.GetCount()
	msg := makeVStateRequestEvent(pn, objectGlobal, payload.RequestLatestDirtyState)
	server.SendMessage(ctx, msg)

	if !server.PublisherMock.WaitCount(countBefore+1, 10*time.Second) {
		t.Fatal("timeout waiting for VStateReport")
	}
}

func TestVirtual_VStateRequest_Unknown(t *testing.T) {
	t.Log("C4863")

	var (
		mc = minimock.NewController(t)
	)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	var (
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
		pn           = server.GetPulse().PulseNumber
	)

	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
		assert.Equal(t, &payload.VStateReport{
			Status: payload.Missing,
			AsOf:   server.GetPrevPulse().PulseNumber,
			Callee: objectGlobal,
		}, report)

		return false
	})

	server.IncrementPulse(ctx)

	countBefore := server.PublisherMock.GetCount()
	msg := makeVStateRequestEvent(pn, objectGlobal, payload.RequestLatestDirtyState)
	server.SendMessage(ctx, msg)

	if !server.PublisherMock.WaitCount(countBefore+1, 10*time.Second) {
		t.Fatal("timeout waiting for VStateReport")
	}
}
