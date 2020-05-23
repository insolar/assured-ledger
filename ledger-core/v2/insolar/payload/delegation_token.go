// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

type TokenType uint16

const (
	Uninitialized TokenType = 0
	DelegateCall  TokenType = 1
	DelegateObjV  TokenType = 2
	DelegateObjL  TokenType = 3
	DelegateDrop  TokenType = 4
)

// CallDelegationToken is delegation tocken for Call Request
type CallDelegationToken struct {
	TokenTypeAndFlags TokenType
	Approver          reference.Global
	DelegateTo        reference.Global
	PulseNumber       pulse.Number
	Callee            reference.Global
	Caller            reference.Global
	Outgoing          reference.Global
	ApproverSignature []byte
}

// IsZero returns true if TokenTypeAndFlags initialized
func (t *CallDelegationToken) IsZero() bool {
	return t.TokenTypeAndFlags == 0
}
