// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package call

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/pulsestor"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

type ID uint64

// LogicContext is a context of contract execution. Everything
// that is required to implement foundation functions. This struct
// shouldn't be used in core components.
type LogicContext struct {
	ID   ID
	Mode Mode // either "execution" or "validation"

	Request reference.Global // reference of incoming request record

	Callee    reference.Global // Contract that is called
	Parent    reference.Global // Parent of the callee
	Prototype reference.Global // Prototype (base class) of the callee
	Code      reference.Global // Code reference of the callee

	Caller          reference.Global // Contract that made the call
	CallerPrototype reference.Global // Prototype (base class) of the caller

	TraceID string          // trace mark for Jaeger and friends
	Pulse   pulsestor.Pulse // pre-fetched pulse for call context
}

type ContractCallType uint8

const (
	_ ContractCallType = iota

	ContractCallOrdered
	ContractCallUnordered
	ContractCallSaga
)

func (t ContractCallType) String() string {
	switch t {
	case ContractCallOrdered:
		return "Ordered"
	case ContractCallUnordered:
		return "Unordered"
	case ContractCallSaga:
		return "Saga"
	default:
		return "Unknown"
	}
}

// Mode indicates whether we execute or validate
type Mode int

const (
	Execute Mode = iota
	Validate
)

func (m Mode) String() string {
	switch m {
	case Execute:
		return "execute"
	case Validate:
		return "validate"
	default:
		return "unknown"
	}
}
