// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package insolar

import (
	"github.com/pkg/errors"
)

func NewCodeDescriptor(code []byte, machineType MachineType, ref Reference) CodeDescriptor {
	return &codeDescriptor{
		code:        code,
		machineType: machineType,
		ref:         ref,
	}
}

// CodeDescriptor represents meta info required to fetch all code data.
type codeDescriptor struct {
	code        []byte
	machineType MachineType
	ref         Reference
}

// Ref returns reference to represented code record.
func (d *codeDescriptor) Ref() *Reference {
	return &d.ref
}

// MachineType returns code machine type for represented code.
func (d *codeDescriptor) MachineType() MachineType {
	return d.machineType
}

// Code returns code data.
func (d *codeDescriptor) Code() ([]byte, error) {
	return d.code, nil
}

// ObjectDescriptor represents meta info required to fetch all object data.
type objectDescriptor struct {
	head      Reference
	state     ID
	prototype *Reference
	memory    []byte
	parent    Reference

	requestID *ID
}

// Prototype returns prototype reference.
func (d *objectDescriptor) Prototype() (*Reference, error) {
	if d.prototype == nil {
		return nil, errors.New("object has no prototype")
	}
	return d.prototype, nil
}

// HeadRef returns reference to represented object record.
func (d *objectDescriptor) HeadRef() *Reference {
	return &d.head
}

// StateID returns reference to object state record.
func (d *objectDescriptor) StateID() *ID {
	return &d.state
}

// Memory fetches latest memory of the object known to storage.
func (d *objectDescriptor) Memory() []byte {
	return d.memory
}

// Parent returns object's parent.
func (d *objectDescriptor) Parent() *Reference {
	return &d.parent
}

func (d *objectDescriptor) EarliestRequestID() *ID {
	return d.requestID
}

func NewPrototypeDescriptor(
	head Reference, state ID, code Reference,
) PrototypeDescriptor {
	return &prototypeDescriptor{
		head:  head,
		state: state,
		code:  code,
	}
}

type prototypeDescriptor struct {
	head  Reference
	state ID
	code  Reference
}

// Code returns code reference.
func (d *prototypeDescriptor) Code() *Reference {
	return &d.code
}

// HeadRef returns reference to represented object record.
func (d *prototypeDescriptor) HeadRef() *Reference {
	return &d.head
}

// StateID returns reference to object state record.
func (d *prototypeDescriptor) StateID() *ID {
	return &d.state
}
