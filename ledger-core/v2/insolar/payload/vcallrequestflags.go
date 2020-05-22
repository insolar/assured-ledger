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
	// 1111111111111100
	interferenceMask = 0xfffc
)

func (f *CallRequestFlags) SetInterference(t contract.InterferenceFlag) {
	if t > contract.LastKnownInterferenceFlag {
		panic(throw.IllegalValue())
	}
	*f = *f&interferenceMask | CallRequestFlags(t)&3
}

func (f CallRequestFlags) GetInterference() contract.InterferenceFlag {
	return contract.InterferenceFlag(f & 3)
}

const (
	// 1111111111110011
	stateMask = 0xfff3
)

func (f *CallRequestFlags) SetState(s contract.StateFlag) {
	if s > contract.LastKnownStateFlag {
		panic(throw.IllegalValue())
	}
	*f = (*f & stateMask) | ((CallRequestFlags(s) & 3) << 2)
}

func (f CallRequestFlags) GetState() contract.StateFlag {
	return contract.StateFlag((f >> 2) & 3)
}

func BuildCallRequestFlags(interference contract.InterferenceFlag, state contract.StateFlag) CallRequestFlags {
	var callFlags CallRequestFlags
	callFlags.SetInterference(interference)
	callFlags.SetState(state)
	return callFlags
}
