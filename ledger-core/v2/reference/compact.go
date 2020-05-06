// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

func Empty() PtrHolder {
	return emptyHolder
}

func EmptyLocal() *Local {
	return &emptyLocal
}

func NewPtrRecord(local Local) PtrHolder {
	if local.IsEmpty() {
		return Empty()
	}
	return NewNoCopy(&local, &emptyLocal)
}

func NewPtrSelf(local Local) PtrHolder {
	if local.IsEmpty() {
		return Empty()
	}
	return compact{&local, &local}
}

func NewPtrHolder(local, base Local) PtrHolder {
	return NewNoCopy(&local, &base)
}

func NewNoCopy(local, base *Local) PtrHolder {
	switch {
	case local.IsEmpty():
		if base.IsEmpty() {
			return Empty()
		}
		local = &emptyLocal
	case base.IsEmpty():
		base = &emptyLocal
	}
	return compact{local, base}
}

var emptyLocal Local
var emptyHolder = compact{&emptyLocal, &emptyLocal}

type compact struct {
	addressLocal *Local
	addressBase  *Local
}

func (v compact) IsZero() bool {
	return v.addressLocal == nil
}

func (v compact) IsEmpty() bool {
	return v.addressLocal.IsEmpty() && v.addressBase.IsEmpty()
}

func (v compact) GetScope() Scope {
	return v.GetBase().SubScope().AsBaseOf(v.GetLocal().SubScope())
}

func (v compact) GetBase() Local {
	if v.addressBase == nil {
		return Local{}
	}
	return *v.addressBase
}

func (v compact) GetLocal() Local {
	if v.addressLocal == nil {
		return Local{}
	}
	return *v.addressLocal
}

func (v compact) GetLocalPtr() *Local {
	return v.addressLocal
}

func (v compact) GetBasePtr() *Local {
	return v.addressBase
}
