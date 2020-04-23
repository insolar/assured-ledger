// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package calltype

type ContractCallType uint8

const (
	_ ContractCallType = iota

	ContractCallOrdered
	ContractCallUnordered
	ContractCallSaga
)

func (t ContractCallType) String() string {
	switch t {
	case ContractCallOrdered:
		return "Ordered"
	case ContractCallUnordered:
		return "Unordered"
	case ContractCallSaga:
		return "Saga"
	default:
		return "Unknown"
	}
}
