// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

func Empty() Holder {
	return emptyHolder
}

func EmptyLocal() *Local {
	return &emptyLocal
}

func NewRecord(local Local) Holder {
	if local.IsEmpty() {
		return Empty()
	}
	return NewNoCopy(&local, &emptyLocal)
}

func NewSelf(local Local) Holder {
	if local.IsEmpty() {
		return Empty()
	}
	return compact{&local, &local}
}

func New(local, base Local) Holder {
	return NewNoCopy(&local, &base)
}

func NewNoCopy(local, base *Local) Holder {
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
	return Scope(v.addressBase.getScope()<<2 | v.addressLocal.getScope())
}

func (v compact) GetBase() *Local {
	return v.addressBase
}

func (v compact) GetLocal() *Local {
	return v.addressLocal
}
