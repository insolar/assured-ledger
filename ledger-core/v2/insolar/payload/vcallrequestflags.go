// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type CallRequestFlags uint32

type StateFlag byte

const (
	CallDirty StateFlag = iota
	CallValidated

	lastKnownStateFlag
)

type ToleranceFlag byte

const (
	CallIntolerable ToleranceFlag = iota
	CallTolerable

	lastKnownToleranceFlag
)

func (f CallRequestFlags) Equal(r CallRequestFlags) bool {
	return f == r
}

const (
	// 1111111111111100
	toleranceMask = 0xfffc
)

func (f *CallRequestFlags) SetTolerance(t ToleranceFlag) {
	if t > lastKnownToleranceFlag {
		panic(throw.IllegalValue())
	}
	*f = *f&toleranceMask | CallRequestFlags(t)&3
}

func (f CallRequestFlags) GetTolerance() ToleranceFlag {
	return ToleranceFlag(f & 3)
}

const (
	// 1111111111110011
	stateMask = 0xfff3
)

func (f *CallRequestFlags) SetState(s StateFlag) {
	if s > lastKnownStateFlag {
		panic(throw.IllegalValue())
	}
	*f = (*f & stateMask) | ((CallRequestFlags(s) & 3) << 2)
}

func (f CallRequestFlags) GetState() StateFlag {
	return StateFlag((f >> 2) & 3)
}
