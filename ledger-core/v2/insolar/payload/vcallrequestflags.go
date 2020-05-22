// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type CallRequestFlags uint32

func (f CallRequestFlags) Equal(r CallRequestFlags) bool {
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

func (f CallRequestFlags) WithInterference(t contract.InterferenceFlag) CallRequestFlags {
	if t > contract.InterferenceFlagCount {
		panic(throw.IllegalValue())
	}
	return (f &^ bitInterferenceMask) | (CallRequestFlags(t) << bitInterferenceFlagOffset)
}

func (f CallRequestFlags) GetInterference() contract.InterferenceFlag {
	return contract.InterferenceFlag(f&bitInterferenceMask) >> bitInterferenceFlagOffset
}

const (
	bitStateFlagMask = ((1 << bitStateFlagCount) - 1) << bitStateFlagOffset
)

func (f CallRequestFlags) WithState(s contract.StateFlag) CallRequestFlags {
	if s > contract.StateFlagCount {
		panic(throw.IllegalValue())
	}
	return (f &^ bitStateFlagMask) | (CallRequestFlags(s) << bitStateFlagOffset)
}

func (f CallRequestFlags) GetState() contract.StateFlag {
	return contract.StateFlag(f&bitStateFlagMask) >> bitStateFlagOffset
}

func BuildCallRequestFlags(interference contract.InterferenceFlag, state contract.StateFlag) CallRequestFlags {
	return CallRequestFlags(0).WithInterference(interference).WithState(state)
}
