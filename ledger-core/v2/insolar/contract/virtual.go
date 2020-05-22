// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package contract

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

// MethodFunc is a typedef for wrapper contract header
type MethodFunc func(oldState []byte, args []byte) (newState []byte, result []byte, err error)

func ConstructorIsolation() MethodIsolation {
	return MethodIsolation{
		Interference: CallTolerable,
		State:        CallDirty,
	}
}

type MethodIsolation struct {
	Interference InterferenceFlag
	State        StateFlag
}

// Method is a struct for Method and it's properties
type Method struct {
	Func      MethodFunc
	Isolation MethodIsolation
}

// Methods maps name to contract method
type Methods map[string]Method

// Constructor is a typedef of typical contract constructor
type Constructor func(ref reference.Global, args []byte) (state []byte, result []byte, err error)

// Constructors maps name to contract constructor
type Constructors map[string]Constructor

// Wrapper stores all needed about contract wrapper (it's methods/constructors)
type Wrapper struct {
	GetCode      MethodFunc
	GetPrototype MethodFunc

	Methods      Methods
	Constructors Constructors
}

type StateFlag byte

const (
	CallDirty StateFlag = iota
	CallValidated

	StateFlagCount = iota
)

type InterferenceFlag byte

const (
	CallIntolerable InterferenceFlag = iota
	CallTolerable

	InterferenceFlagCount = iota
)
