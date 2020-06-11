// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/contract/testwallet"
	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

func makeVStateReportEvent(pulseNumber pulse.Number, objectRef reference.Global, stateRef reference.Local, rawState []byte) *message.Message {
	class := testwalletProxy.GetClass()

	payload := &payload.VStateReport{
		Status: payload.Ready,
		Callee: objectRef,
		ProvidedContent: &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: &payload.ObjectState{
				Reference: stateRef,
				Class:     class,
				State:     rawState,
			},
		},
	}

	return utils.NewRequestWrapper(pulseNumber, payload).Finalize()
}

func makeVStateReportWithState(
	pulseNumber pulse.Number,
	objectRef reference.Global,
	stateStatus payload.VStateReport_StateStatus,
	state *payload.ObjectState,
) *message.Message {
	payload := &payload.VStateReport{
		Status: stateStatus,
		Callee: objectRef,
		ProvidedContent: &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: state,
		},
	}

	return utils.NewRequestWrapper(pulseNumber, payload).Finalize()
}

func makeRawWalletState(balance uint32) []byte {
	return insolar.MustSerialize(testwallet.Wallet{
		Balance: balance,
	})
}

func checkBalance(ctx context.Context, t *testing.T, server *utils.Server, objectRef reference.Global, testBalance uint32) {
	code, byteBuffer := server.CallAPIGetBalance(ctx, objectRef)
	require.Equal(t, 200, code, string(byteBuffer))

	response, err := utils.UnmarshalWalletGetBalanceResponse(byteBuffer)
	require.NoError(t, err)
	require.Empty(t, response.Err)
	require.NotEmpty(t, response.TraceID)
	require.Equal(t, uint(testBalance), response.Amount)
}

func TestVirtual_VStateReport_HappyPath(t *testing.T) {
	t.Log("C4866")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	var (
		testBalance    = uint32(555)
		objectLocal    = server.RandomLocalWithPulse()
		objectGlobal   = reference.NewSelf(objectLocal)
		stateID        = server.RandomLocalWithPulse()
		rawWalletState = makeRawWalletState(testBalance)
		pn             = server.GetPulse().PulseNumber
	)

	{
		// send VStateReport: save wallet
		msg := makeVStateReportEvent(pn, objectGlobal, stateID, rawWalletState)
		server.SendMessage(ctx, msg)
	}

	checkBalance(ctx, t, server, objectGlobal, testBalance)
}

func TestVirtual_VStateReport_TwoStateReports(t *testing.T) {
	t.Log("C4919")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	var (
		testBalance    = uint32(555)
		objectLocal    = server.RandomLocalWithPulse()
		objectGlobal   = reference.NewSelf(objectLocal)
		stateID        = server.RandomLocalWithPulse()
		newStateID     = server.RandomLocalWithPulse()
		rawWalletState = makeRawWalletState(testBalance)
		pn             = server.GetPulse().PulseNumber
	)

	{
		// send VStateReport: save wallet
		msg := makeVStateReportEvent(pn, objectGlobal, stateID, rawWalletState)
		server.SendMessage(ctx, msg)
	}

	checkBalance(ctx, t, server, objectGlobal, testBalance)

	{
		// send VStateReport: one more time to simulate rewrite
		msg := makeVStateReportEvent(pn, objectGlobal, newStateID, makeRawWalletState(444))
		server.SendMessage(ctx, msg)
	}

	checkBalance(ctx, t, server, objectGlobal, testBalance)
}

func TestVirtual_VStateReport_BadState_NoSuchObject(t *testing.T) {
	t.Log("C4864")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	var (
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
	)

	reasons := []payload.VStateReport_StateStatus{payload.Inactive, payload.Missing, payload.Unknown}
	for _, reason := range reasons {
		msg := makeVStateReportWithState(server.GetPulse().PulseNumber, objectGlobal, reason, nil)
		server.SendMessage(ctx, msg)
	}
}

func TestVirtual_VStateReport_BadState_StateAlreadyExists(t *testing.T) {
	t.Log("C4865")

	server, ctx := utils.NewServerIgnoreLogErrors(nil, t)
	defer server.Stop()

	var (
		testBalance    = uint32(555)
		objectLocal    = server.RandomLocalWithPulse()
		objectGlobal   = reference.NewSelf(objectLocal)
		stateID        = server.RandomLocalWithPulse()
		rawWalletState = makeRawWalletState(testBalance)
		pn             = server.GetPulse().PulseNumber
	)
	{
		// send VStateReport: save wallet
		msg := makeVStateReportEvent(pn, objectGlobal, stateID, rawWalletState)
		server.SendMessage(ctx, msg)
	}

	reasons := []payload.VStateReport_StateStatus{payload.Inactive, payload.Missing, payload.Unknown}
	for _, reason := range reasons {
		msg := makeVStateReportWithState(pn, objectGlobal, reason, nil)
		server.SendMessage(ctx, msg)
	}

	checkBalance(ctx, t, server, objectGlobal, testBalance)
}
