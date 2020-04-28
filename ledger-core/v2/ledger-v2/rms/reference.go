// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ reference.Holder = &Reference{}
var _ GoGoMarshaller = &Reference{}

type Reference struct {
	local, base reference.Local
}

func (r *Reference) ProtoSize() int {
	return reference.ProtoSize(r)
}

func (r *Reference) MarshalTo(b []byte) (int, error) {
	return reference.MarshalTo(r, b)
}

func (r *Reference) Unmarshal(b []byte) (err error) {
	r.local, r.base, err = reference.Unmarshal(b)
	return
}

func (r *Reference) GetLocal() *reference.Local {
	return &r.local
}

func (r *Reference) GetBase() *reference.Local {
	return &r.base
}

func (r *Reference) IsEmpty() bool {
	return r.local.IsEmpty() && r.base.IsEmpty()
}

func (r *Reference) GetScope() reference.Scope {
	return r.base.SubScope().AsBaseOf(r.local.SubScope())
}

func (r *Reference) Set(h reference.Holder) {
	r.local, r.base = *h.GetLocal(), *h.GetBase()
}

func (r *Reference) SetLocal(l reference.Local) {
	r.local, r.base = l, reference.Local{}
}

func (r *Reference) Get() reference.Global {
	return reference.NewGlobal(r.base, r.local)
}

func NewLazyRef(base reference.Local, pn pulse.Number, subScope reference.SubScope, hash *HashDispenser) LazyReference {
	switch {
	case hash == nil:
		panic(throw.IllegalValue())
	case !pn.IsTimePulse():
		// NB! special pulses don't have fixed size and can't be used for lazy size calc
		panic(throw.IllegalValue())
	case subScope != 0 && base.IsEmpty():
		panic(throw.IllegalValue())
	}
	return LazyReference{Reference{
		base:  base,
		local: reference.NewLocalTemplate(pn, subScope),
	}, hash}
}

func NewLazySelfRef(pn pulse.Number, hash *HashDispenser) LazyReference {
	switch {
	case hash == nil:
		panic(throw.IllegalValue())
	case !pn.IsTimePulse():
		// NB! special pulses don't have fixed size and can't be used for lazy size calc
		panic(throw.IllegalValue())
	}
	return LazyReference{Reference{
		base:  reference.NewRecordID(pulse.LocalRelative, reference.LocalHash{}),
		local: reference.NewRecordID(pn, reference.LocalHash{}),
	}, hash}
}

func NewLazyLocal(pn pulse.Number, hash *HashDispenser) LazyReference {
	switch {
	case hash == nil:
		panic(throw.IllegalValue())
	case !pn.IsTimePulse():
		// NB! special pulses don't have fixed size and can't be used for lazy size calc
		panic(throw.IllegalValue())
	}
	return LazyReference{Reference{
		local: reference.NewRecordID(pn, reference.LocalHash{}),
	}, hash}
}

var _ reference.Holder = &LazyReference{}
var _ GoGoMarshaller = &LazyReference{}

type LazyReference struct {
	ref  Reference
	hash *HashDispenser
}

func (r *LazyReference) prepare() bool {
	if r.hash == nil {
		return true
	}
	if !r.hash.IsSet() {
		return false
	}
	hash := reference.AsLocalHash(r.hash.value)
	r.hash = nil
	r.ref.local = r.ref.local.WithHash(hash)
	if r.ref.base.Pulse() == pulse.LocalRelative {
		r.ref.base = reference.NewLocal(r.ref.local.Pulse(), r.ref.base.SubScope(), hash)
	}
	return true
}

func (r *LazyReference) ProtoSize() int {
	return r.ref.ProtoSize()
}

func (r *LazyReference) MarshalTo(b []byte) (int, error) {
	if !r.prepare() {
		panic(throw.IllegalState())
	}
	return r.ref.MarshalTo(b)
}

func (r *LazyReference) Unmarshal(b []byte) (err error) {
	r.hash = nil
	return r.ref.Unmarshal(b)
}

func (r *LazyReference) IsEmpty() bool {
	return r.hash == nil && r.ref.IsEmpty()
}

func (r *LazyReference) GetLocal() *reference.Local {
	if !r.prepare() {
		panic(throw.IllegalState())
	}
	return r.ref.GetLocal()
}

func (r *LazyReference) GetBase() *reference.Local {
	if !r.prepare() {
		panic(throw.IllegalState())
	}
	return r.ref.GetBase()
}

//func (r *LazyReference) Set(h reference.Holder) {
//	r.hash = nil
//	r.ref.local, r.ref.base = *h.GetLocal(), *h.GetBase()
//}
//
//func (r *LazyReference) SetLocal(l reference.Local) {
//	r.hash = nil
//	r.ref.local, r.ref.base = l, reference.Local{}
//}

func (r *LazyReference) Get() reference.Global {
	if !r.prepare() {
		panic(throw.IllegalState())
	}
	return reference.NewGlobal(r.ref.base, r.ref.local)
}

func (r *LazyReference) GetScope() reference.Scope {
	return r.ref.GetScope()
}
