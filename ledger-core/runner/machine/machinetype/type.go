// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package machinetype

// Type is a type of virtual machine
type Type int

// Real constants of Type
const (
	Unknown Type = iota
	Builtin
	LastID
)
