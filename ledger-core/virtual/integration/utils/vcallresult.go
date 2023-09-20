package utils

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/rms"
)

func MakeMinimumValidVStateResult(server *Server, returnArgs []byte) *rms.VCallResult {
	return &rms.VCallResult{
		CallType:        rms.CallTypeMethod,
		CallFlags:       rms.BuildCallFlags(isolation.CallIntolerable, isolation.CallDirty),
		Callee:          rms.NewReference(server.RandomGlobalWithPulse()),
		Caller:          rms.NewReference(server.GlobalCaller()),
		ReturnArguments: rms.NewBytes(returnArgs),
		CallOutgoing:    rms.NewReference(server.BuildRandomOutgoingWithPulse()),
		CallIncoming:    rms.NewReference(server.RandomGlobalWithPulse()),
	}
}
