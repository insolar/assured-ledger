// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"context"
	"testing"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/runner/execution"
	"github.com/insolar/assured-ledger/ledger-core/runner/executionupdate"
	"github.com/insolar/assured-ledger/ledger-core/runner/requestresult"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
	"github.com/insolar/assured-ledger/ledger-core/testutils/runner/logicless"
)

type Helper struct {
	server *Server
	class  reference.Global
}

func NewHelper(srv *Server) *Helper {
	return &Helper{
		server: srv,
		class:  gen.UniqueReference(),
	}
}

func (h *Helper) GetObjectClass() reference.Global {
	return h.class
}

func (h *Helper) calculateOutgoing(pl payload.VCallRequest) reference.Global {
	return reference.NewRecordOf(pl.Callee, pl.CallOutgoing)
}

func (h *Helper) CreateObject(ctx context.Context, t *testing.T) reference.Global {
	var (
		pn          = h.server.GetPulse().PulseNumber
		isolation   = contract.ConstructorIsolation()
		plArguments = insolar.MustSerialize([]interface{}{})
	)

	pl := payload.VCallRequest{
		CallType:       payload.CTConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		Caller:         h.server.GlobalCaller(),
		Callee:         h.class,
		CallSiteMethod: "New",
		CallOutgoing:   gen.UniqueIDWithPulse(pn),
		Arguments:      plArguments,
	}
	msg := NewRequestWrapper(pn, &pl).SetSender(h.server.JetCoordinatorMock.Me()).Finalize()
	objectReference := h.calculateOutgoing(pl)

	{
		keyExtractor := func(ctx execution.Context) string { return ctx.Outgoing.String() }
		mockedRunner := logicless.NewServiceMock(ctx, t, keyExtractor)
		h.server.ReplaceRunner(mockedRunner)

		serializedResult := SerializeCreateWalletResultOK(objectReference)

		result := requestresult.New(serializedResult, objectReference)
		result.SetActivate(reference.Global{}, h.class, CreateWallet(initialBalance))

		executionMock := mockedRunner.AddExecutionMock(objectReference.String())
		executionMock.AddStart(nil, &executionupdate.ContractExecutionStateUpdate{
			Type:   executionupdate.Done,
			Result: result,
		})
	}

	typedChecker := h.server.PublisherMock.SetTypedChecker(ctx, t, h.server)
	typedChecker.VCallResult.ExpectedCount(1)

	messagesBefore := h.server.PublisherMock.GetCount()
	h.server.SendMessage(ctx, msg)
	if !h.server.PublisherMock.WaitCount(messagesBefore+1, 10*time.Second) {
		panic("failed to wait for VCallResult")
	}

	return objectReference
}
