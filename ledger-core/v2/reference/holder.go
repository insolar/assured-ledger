// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

type Holder interface {
	GetScope() Scope

	// GetBase returns base portion of a full reference
	GetBase() *Local
	// GetLocal returns local portion of a full reference
	GetLocal() *Local

	IsEmpty() bool
}

func IsRecordScope(ref Holder) bool {
	return ref.GetBase().IsEmpty() && !ref.GetLocal().IsEmpty() && ref.GetLocal().getScope() == baseScopeLifeline
}

func IsObjectReference(ref Holder) bool {
	return !ref.GetBase().IsEmpty() && !ref.GetLocal().IsEmpty() && ref.GetLocal().getScope() == baseScopeLifeline
}

func IsSelfScope(ref Holder) bool {
	return ref.GetBase() == ref.GetLocal() || *ref.GetBase() == *ref.GetLocal()
}

func IsLifelineScope(ref Holder) bool {
	return ref.GetBase().getScope() == baseScopeLifeline
}

func IsLocalDomainScope(ref Holder) bool {
	return ref.GetBase().getScope() == baseScopeLocalDomain
}

func IsGlobalScope(ref Holder) bool {
	return ref.GetBase().getScope() == baseScopeGlobal
}

func Equal(ref0, ref1 Holder) bool {
	if p := ref1.GetLocal(); p == nil || !ref0.GetLocal().Equal(*p) {
		return false
	}
	if p := ref1.GetBase(); p == nil || !ref0.GetBase().Equal(*p) {
		return false
	}
	return true
}

func Compare(ref0, ref1 Holder) int {
	if cmp := ref0.GetBase().Compare(*ref1.GetBase()); cmp != 0 {
		return cmp
	}
	return ref0.GetLocal().Compare(*ref1.GetLocal())
}
