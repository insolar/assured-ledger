// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package isolation

type StateFlag byte

const (
	stateInvalid StateFlag = iota
	CallDirty
	CallValidated

	StateFlagCount = iota
)

func (f StateFlag) IsZero() bool {
	return f == 0
}

func (f StateFlag) IsValid() bool {
	return f > stateInvalid && f < StateFlagCount
}

type InterferenceFlag byte

const (
	interferenceInvalid InterferenceFlag = iota
	CallIntolerable
	CallTolerable

	InterferenceFlagCount = iota
)

func (f InterferenceFlag) IsZero() bool {
	return f == 0
}

func (f InterferenceFlag) IsValid() bool {
	return f > interferenceInvalid && f < InterferenceFlagCount
}
