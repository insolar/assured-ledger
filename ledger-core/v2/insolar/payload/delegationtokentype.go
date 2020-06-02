// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

type CallDelegationTokenType uint16

const (
	DelegationTokenTypeUninitialized CallDelegationTokenType = iota
	DelegationTokenTypeCall
	DelegationTokenTypeObjV
	DelegationTokenTypeObjL
	DelegationTokenTypeDrop
)

func (t CallDelegationTokenType) Equal(other CallDelegationTokenType) bool {
	return t == other
}

func (t CallDelegationToken) IsZero() bool {
	return t.TokenTypeAndFlags == DelegationTokenTypeUninitialized
}
