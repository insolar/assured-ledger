// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package callflag

type CallFlag uint32

const (
	_ CallFlag = 1 << iota
	Unordered
)

func (f CallFlag) Equal(r CallFlag) bool {
	return f == r
}
