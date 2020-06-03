// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
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

func (f CallFlags) WithInterference(t contract.InterferenceFlag) CallFlags {
	if t == 0 {
		panic(throw.IllegalValue())
	}
	if t > contract.InterferenceFlagCount {
		panic(throw.IllegalValue())
	}
	return (f &^ bitInterferenceMask) | (CallFlags(t) << bitInterferenceFlagOffset)
}

func (f CallFlags) GetInterference() contract.InterferenceFlag {
	return contract.InterferenceFlag(f&bitInterferenceMask) >> bitInterferenceFlagOffset
}

const (
	bitStateFlagMask = ((1 << bitStateFlagCount) - 1) << bitStateFlagOffset
)

func (f CallFlags) WithState(s contract.StateFlag) CallFlags {
	if s == 0 {
		panic(throw.IllegalValue())
	}
	if s > contract.StateFlagCount {
		panic(throw.IllegalValue())
	}
	return (f &^ bitStateFlagMask) | (CallFlags(s) << bitStateFlagOffset)
}

func (f CallFlags) GetState() contract.StateFlag {
	return contract.StateFlag(f&bitStateFlagMask) >> bitStateFlagOffset
}

func BuildCallFlags(interference contract.InterferenceFlag, state contract.StateFlag) CallFlags {
	return CallFlags(0).WithInterference(interference).WithState(state)
}
