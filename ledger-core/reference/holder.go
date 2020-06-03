// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

import "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

type LocalHolder interface {
	// GetLocal returns local portion of a full reference
	GetLocal() Local

	IsEmpty() bool
}

type Holder interface {
	LocalHolder

	GetScope() Scope

	// GetBase returns base portion of a full reference
	GetBase() Local
}

type PtrHolder interface {
	Holder

	// GetBase returns base portion of a full reference
	GetLocalPtr() *Local
	GetBasePtr() *Local
}

func AsRecordID(v Holder) Local {
	if IsSelfScope(v) || IsRecordScope(v) {
		return v.GetLocal()
	}
	panic(throw.IllegalState())
}

func IsRecordScope(ref Holder) bool {
	return ref.GetBase().IsEmpty() && !ref.GetLocal().IsEmpty() && ref.GetLocal().SubScope() == baseScopeLifeline
}

func IsObjectReference(ref Holder) bool {
	return !ref.GetBase().IsEmpty() && !ref.GetLocal().IsEmpty() && ref.GetLocal().SubScope() == baseScopeLifeline
}

func IsSelfScope(ref Holder) bool {
	return ref.GetBase() == ref.GetLocal()
}

func IsLifelineScope(ref Holder) bool {
	base := ref.GetBase()
	return base.SubScope() == baseScopeLifeline && !base.IsEmpty()
}

func IsLocalDomainScope(ref Holder) bool {
	return ref.GetBase().SubScope() == baseScopeLocalDomain
}

func IsGlobalScope(ref Holder) bool {
	return ref.GetBase().SubScope() == baseScopeGlobal
}

func Equal(ref0, ref1 Holder) bool {
	switch {
	case ref0 == nil || ref1 == nil:
		return false
	case ref1.GetLocal() != ref0.GetLocal():
		return false
	default:
		return ref1.GetBase() == ref0.GetBase()
	}
}

func Compare(ref0, ref1 Holder) int {
	switch {
	case ref0 == ref1:
		return 0
	case ref0 == nil:
		return -1
	case ref1 == nil:
		return 1
	}
	if cmp := ref0.GetBase().Compare(ref1.GetBase()); cmp != 0 {
		return cmp
	}
	return ref0.GetLocal().Compare(ref1.GetLocal())
}

func Copy(h Holder) Global {
	switch hh := h.(type) {
	case nil:
		return Global{}
	case Global:
		return hh
	default:
		return New(h.GetBase(), h.GetLocal())
	}
}
