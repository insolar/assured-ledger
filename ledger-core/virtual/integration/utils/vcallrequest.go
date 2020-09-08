// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	payload "github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func GenerateVCallRequestConstructor(server *Server) *payload.VCallRequest {
	var (
		isolation = contract.ConstructorIsolation()
	)

	return &payload.VCallRequest{
		CallType:       payload.CallTypeConstructor,
		CallFlags:      payload.BuildCallFlags(isolation.Interference, isolation.State),
		Caller:         server.GlobalCaller(),
		Callee:         gen.UniqueGlobalRefWithPulse(pulse.LocalRelative),
		CallSiteMethod: "New",
		CallSequence:   1,
		CallOutgoing:   server.BuildRandomOutgoingWithPulse(),
		Arguments:      insolar.MustSerialize([]interface{}{}),
	}
}

// GenerateVCallRequestMethod returns CallTypeMethod VCallRequest for tolerable/dirty request by default
func GenerateVCallRequestMethod(server *Server) *payload.VCallRequest {
	return &payload.VCallRequest{
		CallType:       payload.CallTypeMethod,
		CallFlags:      payload.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
		Caller:         server.GlobalCaller(),
		Callee:         server.RandomGlobalWithPulse(),
		CallSiteMethod: "Method",
		CallSequence:   1,
		CallOutgoing:   server.BuildRandomOutgoingWithPulse(),
		Arguments:      insolar.MustSerialize([]interface{}{}),
	}
}

func GenerateVCallRequestMethodImmutable(server *Server) *payload.VCallRequest {
	pl := GenerateVCallRequestMethod(server)
	pl.CallFlags = payload.BuildCallFlags(isolation.CallIntolerable, isolation.CallValidated)

	return pl
}
