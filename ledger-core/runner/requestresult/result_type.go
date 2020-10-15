// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package requestresult

type Type uint8

const (
	SideEffectNone Type = 0
	SideEffectAmend Type = 1<<iota
	SideEffectActivate
	SideEffectDeactivate
)

func (t Type) String() string {
	switch t {
	case SideEffectNone:
		return "None"
	case SideEffectActivate:
		return "Activate"
	case SideEffectAmend:
		return "Amend"
	case SideEffectDeactivate:
		return "Deactivate"
	default:
		return "Unknown"
	}
}
