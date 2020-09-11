// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type CallFlags uint32

func (f CallFlags) Equal(r CallFlags) bool {
	return f == r
}

const (
	bitInterferenceFlagCount = 2
	bitStateFlagCount        = 2

	bitInterferenceFlagOffset = 0
	bitStateFlagOffset        = bitInterferenceFlagOffset + bitInterferenceFlagCount
)

const (
	bitInterferenceMask = ((1 << bitInterferenceFlagCount) - 1) << bitInterferenceFlagOffset
)

func (f CallFlags) WithInterference(t isolation.InterferenceFlag) CallFlags {
	if t == 0 {
		panic(throw.IllegalValue())
	}
	if t > isolation.InterferenceFlagCount {
		panic(throw.IllegalValue())
	}
	return (f &^ bitInterferenceMask) | (CallFlags(t) << bitInterferenceFlagOffset)
}

func (f CallFlags) GetInterference() isolation.InterferenceFlag {
	return isolation.InterferenceFlag(f&bitInterferenceMask) >> bitInterferenceFlagOffset
}

const (
	bitStateFlagMask = ((1 << bitStateFlagCount) - 1) << bitStateFlagOffset
)

func (f CallFlags) WithState(s isolation.StateFlag) CallFlags {
	if s == 0 {
		panic(throw.IllegalValue())
	}
	if s > isolation.StateFlagCount {
		panic(throw.IllegalValue())
	}
	return (f &^ bitStateFlagMask) | (CallFlags(s) << bitStateFlagOffset)
}

func (f CallFlags) GetState() isolation.StateFlag {
	return isolation.StateFlag(f&bitStateFlagMask) >> bitStateFlagOffset
}

func (f CallFlags) IsValid() bool {
	return f.GetInterference().IsValid() && f.GetState().IsValid() && (f&^bitStateFlagMask&^bitInterferenceMask == 0)
}

func BuildCallFlags(interference isolation.InterferenceFlag, state isolation.StateFlag) CallFlags {
	return CallFlags(0).WithInterference(interference).WithState(state)
}
