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
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/testutils/gen"
)

func GenerateVCallRequestConstructor(server *Server) *rms.VCallRequest {
	var (
		isolation = contract.ConstructorIsolation()
	)

	return &rms.VCallRequest{
		CallType:       rms.CallTypeConstructor,
		CallFlags:      rms.BuildCallFlags(isolation.Interference, isolation.State),
		Caller:         server.GlobalCaller(),
		Callee:         gen.UniqueGlobalRefWithPulse(pulse.LocalRelative),
		CallSiteMethod: "New",
		CallSequence:   1,
		CallOutgoing:   server.BuildRandomOutgoingWithPulse(),
		Arguments:      insolar.MustSerialize([]interface{}{}),
	}
}

// GenerateVCallRequestMethod returns CallTypeMethod VCallRequest for tolerable/dirty request by default
func GenerateVCallRequestMethod(server *Server) *rms.VCallRequest {
	return &rms.VCallRequest{
		CallType:       rms.CallTypeMethod,
		CallFlags:      rms.BuildCallFlags(isolation.CallTolerable, isolation.CallDirty),
		Caller:         server.GlobalCaller(),
		Callee:         server.RandomGlobalWithPulse(),
		CallSiteMethod: "Method",
		CallSequence:   1,
		CallOutgoing:   server.BuildRandomOutgoingWithPulse(),
		Arguments:      insolar.MustSerialize([]interface{}{}),
	}
}

func GenerateVCallRequestMethodImmutable(server *Server) *rms.VCallRequest {
	pl := GenerateVCallRequestMethod(server)
	pl.CallFlags = rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallValidated)

	return pl
}
