// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/contract/testwallet"
	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
	"github.com/insolar/assured-ledger/ledger-core/virtual/testutils"
)

func makeVStateReportEvent(objectRef reference.Global, stateRef reference.Local, rawState []byte) *payload.VStateReport {
	class := testwalletProxy.GetClass()

	return &payload.VStateReport{
		Status: payload.Ready,
		Object: objectRef,
		ProvidedContent: &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: &payload.ObjectState{
				Reference: stateRef,
				Class:     class,
				State:     rawState,
			},
		},
	}
}

func makeVStateReportWithState(
	objectRef reference.Global,
	stateStatus payload.VStateReport_StateStatus,
	state *payload.ObjectState,
) *payload.VStateReport {
	res := payload.VStateReport{
		Status: stateStatus,
		Object: objectRef,
	}
	if state != nil {
		res.ProvidedContent = &payload.VStateReport_ProvidedContentBody{
			LatestDirtyState: state,
		}
	}
	return &res

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
	defer commontestutils.LeakTester(t)

	t.Log("C4866")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()
	server.IncrementPulse(ctx)

	var (
		testBalance    = uint32(555)
		objectLocal    = server.RandomLocalWithPulse()
		objectGlobal   = reference.NewSelf(objectLocal)
		stateID        = server.RandomLocalWithPulse()
		rawWalletState = makeRawWalletState(testBalance)
	)

	{
		// send VStateReport: save wallet

		pl := makeVStateReportEvent(objectGlobal, stateID, rawWalletState)
		server.SendPayload(ctx, pl)
	}

	checkBalance(ctx, t, server, objectGlobal, testBalance)
}

func TestVirtual_VStateReport_TwoStateReports(t *testing.T) {
	defer commontestutils.LeakTester(t)

	t.Log("C4919")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()
	server.IncrementPulse(ctx)

	var (
		testBalance    = uint32(555)
		objectLocal    = server.RandomLocalWithPulse()
		objectGlobal   = reference.NewSelf(objectLocal)
		stateID        = server.RandomLocalWithPulse()
		newStateID     = server.RandomLocalWithPulse()
		rawWalletState = makeRawWalletState(testBalance)
	)

	{
		// send VStateReport: save wallet
		pl := makeVStateReportEvent(objectGlobal, stateID, rawWalletState)
		server.SendPayload(ctx, pl)
	}

	checkBalance(ctx, t, server, objectGlobal, testBalance)

	{
		// send VStateReport: one more time to simulate rewrite
		pl := makeVStateReportEvent(objectGlobal, newStateID, makeRawWalletState(444))
		server.SendPayload(ctx, pl)
	}

	checkBalance(ctx, t, server, objectGlobal, testBalance)
}

func TestVirtual_VStateReport_BadState_NoSuchObject(t *testing.T) {
	defer commontestutils.LeakTester(t)

	t.Log("C4864")

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()
	server.IncrementPulse(ctx)

	var (
		objectLocal  = server.RandomLocalWithPulse()
		objectGlobal = reference.NewSelf(objectLocal)
	)

	reasons := []payload.VStateReport_StateStatus{payload.Inactive, payload.Missing, payload.Unknown}
	for _, reason := range reasons {
		pl := makeVStateReportWithState(objectGlobal, reason, nil)
		server.SendPayload(ctx, pl)
	}
}

func TestVirtual_VStateReport_BadState_StateAlreadyExists(t *testing.T) {
	defer commontestutils.LeakTester(t)

	t.Log("C4865")

	server, ctx := utils.NewServerWithErrorFilter(nil, t, func(s string) bool {
		// Pass all errors, except for (*SMVStateReport).stepProcess
		return !strings.Contains(s, "(*SMVStateReport).stepProcess")
	})
	defer server.Stop()
	server.IncrementPulse(ctx)

	var (
		testBalance    = uint32(555)
		objectLocal    = server.RandomLocalWithPulse()
		objectGlobal   = reference.NewSelf(objectLocal)
		stateID        = server.RandomLocalWithPulse()
		rawWalletState = makeRawWalletState(testBalance)
	)
	{
		// send VStateReport: save wallet
		pl := makeVStateReportEvent(objectGlobal, stateID, rawWalletState)
		server.SendPayload(ctx, pl)
	}

	reasons := []payload.VStateReport_StateStatus{payload.Inactive, payload.Missing, payload.Unknown}
	for _, reason := range reasons {
		pl := makeVStateReportWithState(objectGlobal, reason, nil)
		server.SendPayload(ctx, pl)
	}

	checkBalance(ctx, t, server, objectGlobal, testBalance)
}

func TestVirtual_VStateReport_CheckValidatedState(t *testing.T) {
	defer commontestutils.LeakTester(t)

	t.Log("C5124")

	mc := minimock.NewController(t)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	server.IncrementPulseAndWaitIdle(ctx)

	var (
		objectLocal          = server.RandomLocalWithPulse()
		objectGlobal         = reference.NewSelf(objectLocal)
		dirtyWalletState     = makeRawWalletState(initialBalance)
		validatedWalletState = makeRawWalletState(initialBalance + 123)
	)

	content := &payload.VStateReport_ProvidedContentBody{
		LatestDirtyState: &payload.ObjectState{
			Reference: server.RandomLocalWithPulse(),
			Class:     testwalletProxy.GetClass(),
			State:     dirtyWalletState,
		},
		LatestValidatedState: &payload.ObjectState{
			Reference: server.RandomLocalWithPulse(),
			Class:     testwalletProxy.GetClass(),
			State:     validatedWalletState,
		},
	}

	{
		payload := &payload.VStateReport{
			Status:          payload.Ready,
			Object:          objectGlobal,
			ProvidedContent: content,
		}

		wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
		server.SendPayload(ctx, payload)
		testutils.WaitSignalsTimed(t, 10*time.Second, wait)
	}

	waitVStateReport := make(chan struct{})
	typedChecker := server.PublisherMock.SetTypedChecker(ctx, mc, server)
	typedChecker.VStateReport.Set(func(report *payload.VStateReport) bool {
		require.Equal(t, report.ProvidedContent.LatestDirtyState, content.LatestDirtyState)
		require.Equal(t, report.ProvidedContent.LatestValidatedState, content.LatestDirtyState)
		waitVStateReport <- struct{}{}
		return false
	})

	server.IncrementPulseAndWaitIdle(ctx)
	testutils.WaitSignalsTimed(t, 10*time.Second, waitVStateReport)

	mc.Finish()
}
