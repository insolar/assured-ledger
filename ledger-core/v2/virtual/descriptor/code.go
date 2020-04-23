// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package descriptor

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/insolar"
)

// CodeDescriptor represents meta info required to fetch all code data.
type CodeDescriptor interface {
	// Ref returns reference to represented code record.
	Ref() *insolar.Reference

	// MachineType returns code machine type for represented code.
	MachineType() insolar.MachineType

	// Code returns code data.
	Code() ([]byte, error)
}

func NewCodeDescriptor(code []byte, machineType insolar.MachineType, ref insolar.Reference) CodeDescriptor {
	return &codeDescriptor{
		code:        code,
		machineType: machineType,
		ref:         ref,
	}
}

// CodeDescriptor represents meta info required to fetch all code data.
type codeDescriptor struct {
	code        []byte
	machineType insolar.MachineType
	ref         insolar.Reference
}

// Ref returns reference to represented code record.
func (d *codeDescriptor) Ref() *insolar.Reference {
	return &d.ref
}

// MachineType returns code machine type for represented code.
func (d *codeDescriptor) MachineType() insolar.MachineType {
	return d.machineType
}

// Code returns code data.
func (d *codeDescriptor) Code() ([]byte, error) {
	return d.code, nil
}
