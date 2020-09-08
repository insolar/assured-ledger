// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

func MakeMinimumValidVStateResult(server *Server, returnArgs []byte) *rms.VCallResult {
	return &rms.VCallResult{
		CallType:        rms.CallTypeMethod,
		CallFlags:       rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
		Callee:          server.RandomGlobalWithPulse(),
		Caller:          server.GlobalCaller(),
		ReturnArguments: returnArgs,
		CallOutgoing:    server.BuildRandomOutgoingWithPulse(),
		CallIncoming:    server.RandomGlobalWithPulse(),
	}
}
