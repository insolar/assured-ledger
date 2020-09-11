// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

func GetSenderDelegationToken(msg interface{}) (CallDelegationToken, bool) {
	type tokenHolder interface {
		GetDelegationSpec() CallDelegationToken
	}

	if th, ok := msg.(tokenHolder); ok {
		return th.GetDelegationSpec(), true
	}
	return CallDelegationToken{}, false
}
