// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

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

func (m CallDelegationToken) IsZero() bool {
	return m.TokenTypeAndFlags == DelegationTokenTypeUninitialized
}
