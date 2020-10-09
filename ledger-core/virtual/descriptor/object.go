// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package descriptor

import (
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/reference"
)

// Object represents meta info about object's particular state.
type Object interface {
	// IsEmpty returns true if descriptor is empty
	IsEmpty() bool

	// HeadRef returns head reference to represented object record.
	HeadRef() reference.Global

	// State returns reference to object state record.
	State() reference.Global

	// Memory fetches object memory from storage.
	Memory() []byte

	// Class returns class reference.
	Class() reference.Global

	// Deactivated return true after SelfDestruct was called
	Deactivated() bool
}

func NewObject(
	head reference.Global,
	state reference.Local,
	class reference.Global,
	memory []byte,
	deactivated bool,
) Object {
	res := object{
		head:        head,
		class:       class,
		memory:      memory,
		deactivated: deactivated,
	}
	if !head.IsEmpty() {
		if state.IsEmpty() {
			panic(errors.IllegalValue())
		}
		res.state = reference.NewRecordOf(head, state)
	} else if !state.IsEmpty() {
		panic(errors.IllegalValue())
	}
	return &res
}

// Object represents meta info required to fetch all object data.
type object struct {
	head        reference.Global
	state       reference.Global
	class       reference.Global
	memory      []byte
	deactivated bool
}

// IsEmpty returns true if descriptor is empty
func (d object) IsEmpty() bool {
	return d.head.IsEmpty()
}

// Class returns class reference.
func (d object) Class() reference.Global {
	return d.class
}

// HeadRef returns reference to represented object record.
func (d object) HeadRef() reference.Global {
	return d.head
}

// State returns reference to object state record.
func (d object) State() reference.Global {
	return d.state
}

// Memory returns memory blob of the object referenced by the State.
func (d object) Memory() []byte {
	return d.memory
}

// Deactivated returns true if object's state is deactivated
func (d object) Deactivated() bool {
	return d.deactivated
}
