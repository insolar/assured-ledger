// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package descriptor

import (
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/reference"
)

// Object represents meta info required to fetch all object data.
type Object interface {
	// HeadRef returns head reference to represented object record.
	HeadRef() reference.Global

	// StateID returns reference to object state record.
	StateID() reference.Local

	// Memory fetches object memory from storage.
	Memory() []byte

	// Class returns class reference.
	Class() (reference.Global, error)

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
	return &object{
		head:        head,
		state:       state,
		class:       class,
		memory:      memory,
		deactivated: deactivated,
	}
}

// Object represents meta info required to fetch all object data.
type object struct {
	head        reference.Global
	state       reference.Local
	class       reference.Global
	memory      []byte
	deactivated bool
}

// Class returns class reference.
func (d object) Class() (reference.Global, error) {
	if d.class.IsEmpty() {
		return reference.Global{}, errors.New("object has no class")
	}
	return d.class, nil
}

// HeadRef returns reference to represented object record.
func (d object) HeadRef() reference.Global {
	return d.head
}

// StateID returns reference to object state record.
func (d object) StateID() reference.Local {
	return d.state
}

// Memory fetches latest memory of the object known to storage.
func (d object) Memory() []byte {
	return d.memory
}

func (d object) Deactivated() bool {
	return d.deactivated
}
