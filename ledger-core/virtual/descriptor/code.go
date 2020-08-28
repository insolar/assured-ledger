// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package descriptor

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
	_type "github.com/insolar/assured-ledger/ledger-core/runner/machine/type"
)

// Code represents meta info required to fetch all code data.
type Code interface {
	// Ref returns reference to represented code record.
	Ref() reference.Global

	// Type returns code machine type for represented code.
	MachineType() _type.Type

	// Code returns code data.
	Code() ([]byte, error)
}

func NewCode(content []byte, machineType _type.Type, ref reference.Global) Code {
	return &code{
		code:        content,
		machineType: machineType,
		ref:         ref,
	}
}

// Code represents meta info required to fetch all code data.
type code struct {
	code        []byte
	machineType _type.Type
	ref         reference.Global
}

// Ref returns reference to represented code record.
func (d *code) Ref() reference.Global {
	return d.ref
}

// Type returns code machine type for represented code.
func (d *code) MachineType() _type.Type {
	return d.machineType
}

// Code returns code data.
func (d *code) Code() ([]byte, error) {
	return d.code, nil
}
