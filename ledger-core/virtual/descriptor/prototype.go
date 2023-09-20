package descriptor

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
)

// Class represents meta info required to fetch all class data.
type Class interface {
	// HeadRef returns head reference to represented object record.
	HeadRef() reference.Global

	// StateID returns reference to object state record.
	StateID() reference.Local

	// Code returns code reference.
	Code() reference.Global
}

func NewClass(
	head reference.Global, state reference.Local, code reference.Global,
) Class {
	return &class{
		head:  head,
		state: state,
		code:  code,
	}
}

type class struct {
	head  reference.Global
	state reference.Local
	code  reference.Global
}

// Code returns code reference.
func (d *class) Code() reference.Global {
	return d.code
}

// HeadRef returns reference to represented object record.
func (d *class) HeadRef() reference.Global {
	return d.head
}

// StateID returns reference to object state record.
func (d *class) StateID() reference.Local {
	return d.state
}
