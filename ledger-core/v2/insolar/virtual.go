// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insolar

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

// CallMode indicates whether we execute or validate
type CallMode int

const (
	ExecuteCallMode CallMode = iota
	ValidateCallMode
)

func (m CallMode) String() string {
	switch m {
	case ExecuteCallMode:
		return "execute"
	case ValidateCallMode:
		return "validate"
	default:
		return "unknown"
	}
}

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

//go:generate stringer -type=PendingState

// PendingState is a state of execution for each object
type PendingState int

const (
	PendingUnknown PendingState = iota // PendingUnknown signalizes that we don't know about execution state
	NotPending                         // NotPending means that we know that this task is not executed by another VE
	InPending                          // InPending means that we know that method on object is executed by another VE
)

func (s PendingState) Equal(other PendingState) bool {
	return s == other
}

type RequestResultType uint8

const (
	RequestSideEffectNone RequestResultType = iota
	RequestSideEffectActivate
	RequestSideEffectAmend
	RequestSideEffectDeactivate
)

func (t RequestResultType) String() string {
	switch t {
	case RequestSideEffectNone:
		return "None"
	case RequestSideEffectActivate:
		return "Activate"
	case RequestSideEffectAmend:
		return "Amend"
	case RequestSideEffectDeactivate:
		return "Deactivate"
	default:
		return "Unknown"
	}
}
