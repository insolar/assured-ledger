// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insolar

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

// ContractMethodFunc is a typedef for wrapper contract header
type ContractMethodFunc func(oldState []byte, args []byte) (newState []byte, result []byte, err error)

// ContractMethod is a struct for Method and it's properties
type ContractMethod struct {
	Func      ContractMethodFunc
	Unordered bool
}

// ContractMethods maps name to contract method
type ContractMethods map[string]ContractMethod

// ContractConstructor is a typedef of typical contract constructor
type ContractConstructor func(ref reference.Global, args []byte) (state []byte, result []byte, err error)

// ContractConstructors maps name to contract constructor
type ContractConstructors map[string]ContractConstructor

// ContractWrapper stores all needed about contract wrapper (it's methods/constructors)
type ContractWrapper struct {
	GetCode      ContractMethodFunc
	GetPrototype ContractMethodFunc

	Methods      ContractMethods
	Constructors ContractConstructors
}

