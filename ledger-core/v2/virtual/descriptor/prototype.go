// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package descriptor

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

// Prototype represents meta info required to fetch all prototype data.
type Prototype interface {
	// HeadRef returns head reference to represented object record.
	HeadRef() reference.Global

	// StateID returns reference to object state record.
	StateID() reference.Local

	// Code returns code reference.
	Code() reference.Global
}

func NewPrototype(
	head reference.Global, state reference.Local, code reference.Global,
) Prototype {
	return &prototype{
		head:  head,
		state: state,
		code:  code,
	}
}

type prototype struct {
	head  reference.Global
	state reference.Local
	code  reference.Global
}

// Code returns code reference.
func (d *prototype) Code() reference.Global {
	return d.code
}

// HeadRef returns reference to represented object record.
func (d *prototype) HeadRef() reference.Global {
	return d.head
}

// StateID returns reference to object state record.
func (d *prototype) StateID() reference.Local {
	return d.state
}
