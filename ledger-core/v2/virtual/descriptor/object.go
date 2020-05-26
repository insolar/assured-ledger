// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package descriptor

import (
	errors "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

// Object represents meta info required to fetch all object data.
type Object interface {
	// HeadRef returns head reference to represented object record.
	HeadRef() reference.Global

	// StateID returns reference to object state record.
	StateID() reference.Local

	// Memory fetches object memory from storage.
	Memory() []byte

	// Prototype returns prototype reference.
	Prototype() (reference.Global, error)

	// Parent returns object's parent.
	Parent() reference.Global
}

func NewObject(
	head reference.Global,
	state reference.Local,
	prototype reference.Global,
	memory []byte,
	parent reference.Global,
) Object {
	return &object{
		head:      head,
		state:     state,
		prototype: prototype,
		memory:    memory,
		parent:    parent,
	}
}

// Object represents meta info required to fetch all object data.
type object struct {
	head      reference.Global
	state     reference.Local
	prototype reference.Global
	memory    []byte
	parent    reference.Global
}

// Prototype returns prototype reference.
func (d *object) Prototype() (reference.Global, error) {
	if d.prototype.IsEmpty() {
		return reference.Global{}, errors.New("object has no prototype")
	}
	return d.prototype, nil
}

// HeadRef returns reference to represented object record.
func (d *object) HeadRef() reference.Global {
	return d.head
}

// StateID returns reference to object state record.
func (d *object) StateID() reference.Local {
	return d.state
}

// Memory fetches latest memory of the object known to storage.
func (d *object) Memory() []byte {
	return d.memory
}

// Parent returns object's parent.
func (d *object) Parent() reference.Global {
	return d.parent
}
