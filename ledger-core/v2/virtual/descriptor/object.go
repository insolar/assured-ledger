// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package descriptor

import (
	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

// ObjectDescriptor represents meta info required to fetch all object data.
type ObjectDescriptor interface {
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

	// EarliestRequestID returns latest requestID for this object
	EarliestRequestID() reference.Local
}

func NewObjectDescriptor(
	head reference.Global, state reference.Local, prototype reference.Global, memory []byte, parent reference.Global, requestID reference.Local,
) ObjectDescriptor {
	return &objectDescriptor{
		head:      head,
		state:     state,
		prototype: prototype,
		memory:    memory,
		parent:    parent,
		requestID: requestID,
	}
}

// ObjectDescriptor represents meta info required to fetch all object data.
type objectDescriptor struct {
	head      reference.Global
	state     reference.Local
	prototype reference.Global
	memory    []byte
	parent    reference.Global

	requestID reference.Local
}

// Prototype returns prototype reference.
func (d *objectDescriptor) Prototype() (reference.Global, error) {
	if d.prototype.IsEmpty() {
		return reference.Global{}, errors.New("object has no prototype")
	}
	return d.prototype, nil
}

// HeadRef returns reference to represented object record.
func (d *objectDescriptor) HeadRef() reference.Global {
	return d.head
}

// StateID returns reference to object state record.
func (d *objectDescriptor) StateID() reference.Local {
	return d.state
}

// Memory fetches latest memory of the object known to storage.
func (d *objectDescriptor) Memory() []byte {
	return d.memory
}

// Parent returns object's parent.
func (d *objectDescriptor) Parent() reference.Global {
	return d.parent
}

func (d *objectDescriptor) EarliestRequestID() reference.Local {
	return d.requestID
}
