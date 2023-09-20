package call

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

type ID uint64

// LogicContext is a context of contract execution. Everything
// that is required to implement foundation functions. This struct
// shouldn't be used in core components.
type LogicContext struct {
	ID   ID
	Mode Mode // either "execution" or "validation"

	Request reference.Global // reference of incoming request record

	Callee reference.Global // Contract that is called
	Parent reference.Global // Parent of the callee
	Class  reference.Global // Class (base class) of the callee
	Code   reference.Global // Code reference of the callee

	Caller      reference.Global // Contract that made the call
	CallerClass reference.Global // Class (base class) of the caller

	TraceID string     // trace mark for Jaeger and friends
	Pulse   pulse.Data // pre-fetched pulse for call context
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
