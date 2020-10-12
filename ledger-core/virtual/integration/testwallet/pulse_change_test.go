// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package testwallet

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/application/builtin/contract/testwallet"
	testwalletProxy "github.com/insolar/assured-ledger/ledger-core/application/builtin/proxy/testwallet"
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	commontestutils "github.com/insolar/assured-ledger/ledger-core/testutils"
	"github.com/insolar/assured-ledger/ledger-core/testutils/insrail"
	"github.com/insolar/assured-ledger/ledger-core/virtual/handlers"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/utils"
)

const initialBalance uint32 = 500

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

func TestVirtual_CallMethodAfterMultiplePulseChanges(t *testing.T) {
	defer commontestutils.LeakTester(t)
	insrail.LogCase(t, "C4918")

	mc := minimock.NewController(t)

	server, ctx := utils.NewServer(nil, t)
	defer server.Stop()

	typedChecker := server.PublisherMock.SetTypedCheckerWithLightStubs(ctx, mc, server)
	typedChecker.VCallRequest.SetResend(true)
	typedChecker.VCallResult.SetResend(true)
	typedChecker.VStateReport.SetResend(true)
	typedChecker.VStateRequest.SetResend(true)
	typedChecker.VObjectTranscriptReport.SetResend(true)

	server.IncrementPulseAndWaitIdle(ctx)

	var (
		objectGlobal    = server.RandomGlobalWithPulse()
		numPulseChanges = 5
	)

	report := server.StateReportBuilder().Object(objectGlobal).Ready().
		Class(testwalletProxy.GetClass()).Memory(makeRawWalletState(initialBalance)).Report()

	server.IncrementPulseAndWaitIdle(ctx)

	{
		wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
		server.SendPayload(ctx, &report)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, wait)
	}

	for i := 0; i < numPulseChanges; i++ {
		wait := server.Journal.WaitStopOf(&handlers.SMVStateReport{}, 1)
		server.IncrementPulseAndWaitIdle(ctx)
		commontestutils.WaitSignalsTimed(t, 10*time.Second, wait)

		checkBalance(ctx, t, server, objectGlobal, initialBalance)
	}

	{
		assert.Equal(t, numPulseChanges, typedChecker.VCallRequest.Count())
		assert.Equal(t, numPulseChanges, typedChecker.VCallResult.Count())
		assert.Equal(t, numPulseChanges, typedChecker.VStateReport.Count())
	}

	mc.Finish()
}
