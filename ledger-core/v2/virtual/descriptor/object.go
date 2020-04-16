// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package descriptor

import (
	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

// ObjectDescriptor represents meta info required to fetch all object data.
type ObjectDescriptor interface {
	// HeadRef returns head reference to represented object record.
	HeadRef() *insolar.Reference

	// StateID returns reference to object state record.
	StateID() *insolar.ID

	// Memory fetches object memory from storage.
	Memory() []byte

	// Prototype returns prototype reference.
	Prototype() (*insolar.Reference, error)

	// Parent returns object's parent.
	Parent() *insolar.Reference

	// EarliestRequestID returns latest requestID for this object
	EarliestRequestID() *insolar.ID
}

func NewObjectDescriptor(
	head insolar.Reference, state insolar.ID, prototype *insolar.Reference, memory []byte, parent insolar.Reference, requestID *insolar.ID,
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
	head      insolar.Reference
	state     insolar.ID
	prototype *insolar.Reference
	memory    []byte
	parent    insolar.Reference

	requestID *insolar.ID
}

// Prototype returns prototype reference.
func (d *objectDescriptor) Prototype() (*insolar.Reference, error) {
	if d.prototype == nil {
		return nil, errors.New("object has no prototype")
	}
	return d.prototype, nil
}

// HeadRef returns reference to represented object record.
func (d *objectDescriptor) HeadRef() *insolar.Reference {
	return &d.head
}

// StateID returns reference to object state record.
func (d *objectDescriptor) StateID() *insolar.ID {
	return &d.state
}

// Memory fetches latest memory of the object known to storage.
func (d *objectDescriptor) Memory() []byte {
	return d.memory
}

// Parent returns object's parent.
func (d *objectDescriptor) Parent() *insolar.Reference {
	return &d.parent
}

func (d *objectDescriptor) EarliestRequestID() *insolar.ID {
	return d.requestID
}
