// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package token

import (
	"context"

	"github.com/insolar/assured-ledger/ledger-core/insolar/payload"
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

var deadBeef = [...]byte{0xde, 0xad, 0xbe, 0xef}

type Service interface {
	GetCallDelegationToken(to reference.Global, pn pulse.Number, object reference.Global) payload.CallDelegationToken
}

type service struct {
	selfNode reference.Global
}

func NewService(_ context.Context, selfNode reference.Global) Service {
	return service{selfNode: selfNode}
}

func (s service) GetCallDelegationToken(to reference.Global, pn pulse.Number, object reference.Global) payload.CallDelegationToken {
	return payload.CallDelegationToken{
		TokenTypeAndFlags: payload.DelegationTokenTypeCall,
		Approver:          s.selfNode,
		DelegateTo:        to,
		PulseNumber:       pn,
		Callee:            object,
		Caller:            to,
		ApproverSignature: deadBeef[:],
	}
}
