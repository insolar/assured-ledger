// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
)

func MakeMinimumValidVStateResult(server *Server, returnArgs []byte) *payload.VCallResult {
	return &payload.VCallResult{
		CallType:        payload.CallTypeMethod,
		CallFlags:       payload.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
		Callee:          server.RandomGlobalWithPulse(),
		Caller:          server.GlobalCaller(),
		ReturnArguments: returnArgs,
		CallOutgoing:    server.BuildRandomOutgoingWithPulse(),
		CallIncoming:    server.RandomGlobalWithPulse(),
	}
}
