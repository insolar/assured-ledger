// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

type LocalHolder interface {
	// GetLocal returns local portion of a full reference
	GetLocal() *Local

	IsEmpty() bool
}

type Holder interface {
	LocalHolder

	GetScope() Scope

	// GetBase returns base portion of a full reference
	GetBase() *Local
}

func IsRecordScope(ref Holder) bool {
	return ref.GetBase().IsEmpty() && !ref.GetLocal().IsEmpty() && ref.GetLocal().SubScope() == baseScopeLifeline
}

func IsObjectReference(ref Holder) bool {
	return !ref.GetBase().IsEmpty() && !ref.GetLocal().IsEmpty() && ref.GetLocal().SubScope() == baseScopeLifeline
}

func IsSelfScope(ref Holder) bool {
	return ref.GetBase() == ref.GetLocal() || *ref.GetBase() == *ref.GetLocal()
}

func IsLifelineScope(ref Holder) bool {
	return ref.GetBase().SubScope() == baseScopeLifeline
}

func IsLocalDomainScope(ref Holder) bool {
	return ref.GetBase().SubScope() == baseScopeLocalDomain
}

func IsGlobalScope(ref Holder) bool {
	return ref.GetBase().SubScope() == baseScopeGlobal
}

func Equal(ref0, ref1 Holder) bool {
	if p := ref1.GetLocal(); p == nil || !p.EqualPtr(ref0.GetLocal()) {
		return false
	}
	if p := ref1.GetBase(); p == nil || !p.EqualPtr(ref0.GetBase()) {
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

func ReadTo(h Holder, b []byte) (int, error) {
	n, err := h.GetLocal().Read(b)
	if err != nil || n < LocalBinarySize {
		return n, err
	}

	n2 := 0
	n2, err = h.GetBase().Read(b[n:])
	return n + n2, err
}
