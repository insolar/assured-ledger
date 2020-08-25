// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package utils

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

func MakeMinimumValidVStateResult(server *Server, returnArgs []byte) *payload.VCallResult {
	return &payload.VCallResult{
		CallType:        payload.CallTypeMethod,
		CallFlags:       payload.BuildCallFlags(contract.CallIntolerable, contract.CallDirty),
		Callee:          reference.NewSelf(server.RandomLocalWithPulse()),
		Caller:          server.GlobalCaller(),
		ReturnArguments: returnArgs,
		CallOutgoing:    server.BuildRandomOutgoingWithPulse(),
		CallIncoming:    server.RandomGlobalWithPulse(),
	}
}
