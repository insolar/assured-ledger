// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/atomickit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type HashDispenser struct {
	state atomickit.OnceFlag
	value cryptkit.Digest
}

func (v *HashDispenser) Get() cryptkit.Digest {
	if v.IsSet() {
		return v.value
	}
	return cryptkit.Digest{}
}

func (v *HashDispenser) IsSet() bool {
	return v != nil && v.state.IsSet()
}

func (v *HashDispenser) set(value cryptkit.Digest) {
	if !v.state.DoSet(func() {
		v.value = value
	}) {
		panic(throw.IllegalState())
	}
}

var _ reference.LocalHolder = &LazyLocalRef{}
var _ GoGoMarshaller = &LazyLocalRef{}

func NewLazyLocalRef(pn pulse.Number, hash *HashDispenser) LazyLocalRef {
	switch {
	case hash == nil:
		panic(throw.IllegalState())
	case !pn.IsSpecialOrTimePulse():
		panic(throw.IllegalValue())
	}
	return LazyLocalRef{reference.NewRecordID(pn, reference.LocalHash{}), hash}
}

type LazyLocalRef struct {
	local reference.Local
	hash  *HashDispenser
}

func (p *LazyLocalRef) ProtoSize() int {
	return p.local.ProtoSize()
}

func (p *LazyLocalRef) MarshalTo(b []byte) (int, error) {
	return p.local.MarshalTo(b)
}

func (p *LazyLocalRef) Marshal() ([]byte, error) {
	return p.local.Marshal()
}

func (p *LazyLocalRef) Unmarshal(b []byte) error {
	p.hash = nil
	return p.local.Unmarshal(b)
}

func (p *LazyLocalRef) getLocal() *reference.Local {
	switch {
	case p.local.IsEmpty():
		return nil
	case p.hash != nil:
		d := p.hash.Get()
		if d.IsEmpty() {
			return nil
		}
		p.hash = nil
		h := reference.AsLocalHash(d)
		p.local = reference.NewRecordID(p.local.Pulse(), h)
		fallthrough
	default:
		return &p.local
	}
}

func (p *LazyLocalRef) GetLocal() *reference.Local {
	if r := p.getLocal(); r != nil {
		return nil
	}
	panic(throw.IllegalState())
}

func (p *LazyLocalRef) IsEmpty() bool {
	return p.getLocal() != nil
}

func NewLazyReference(base reference.Local, pn pulse.Number, subScope reference.SubScope, hash *HashDispenser) LazyGlobalRef {
	switch {
	case hash == nil:
		panic(throw.IllegalState())
	case !pn.IsSpecialOrTimePulse():
		panic(throw.IllegalValue())
	}
	return LazyGlobalRef{base,
		LazyLocalRef{reference.NewLocal(pn, subScope, reference.LocalHash{}), hash}}
}

type lazyLocalRef = LazyLocalRef

var _ reference.Holder = &LazyGlobalRef{}
var _ GoGoMarshaller = &LazyGlobalRef{}

type LazyGlobalRef struct {
	base  reference.Local
	local lazyLocalRef
}

func (p *LazyGlobalRef) ProtoSize() int {
	return reference.GlobalBinarySize
}

func (p *LazyGlobalRef) MarshalTo(dAtA []byte) (int, error) {
	panic(throw.NotImplemented())
	//n, err := p.local.local.Read(p)
	//if err != nil || n < v.addressLocal.len() {
	//	return n, err
	//}
	//n2, err := v.addressBase.Read(p[n:])
	//return n + n2, err
}

func (p *LazyGlobalRef) Marshal() ([]byte, error) {
	panic(throw.NotImplemented())
}

func (p *LazyGlobalRef) Unmarshal(b []byte) error {
	g := reference.Global{}
	if err := g.Unmarshal(b); err != nil {
		return err
	}
	p.local.hash = nil
	p.base = *g.GetBase()
	p.local.local = *g.GetLocal()
	return nil
}

func (p *LazyGlobalRef) GetLocal() *reference.Local {
	return p.local.GetLocal()
}

func (p *LazyGlobalRef) IsEmpty() bool {
	return p.local.IsEmpty()
}

func (p *LazyGlobalRef) GetBase() *reference.Local {
	if !p.local.IsEmpty() {
		return &p.base
	}
	panic(throw.IllegalState())
}

func (p *LazyGlobalRef) GetScope() reference.Scope {
	return p.base.SubScope().AsBaseOf(p.local.local.SubScope())
}

func NewLazySelfReference(pn pulse.Number, subScope reference.SubScope, hash *HashDispenser) LazySelfRef {
	switch {
	case hash == nil:
		panic(throw.IllegalState())
	case !pn.IsSpecialOrTimePulse():
		panic(throw.IllegalValue())
	}
	return LazySelfRef{LazyLocalRef{reference.NewLocal(pn, subScope, reference.LocalHash{}), hash}}
}

var _ reference.Holder = &LazySelfRef{}

type LazySelfRef struct {
	lazyLocalRef
}

func (p *LazySelfRef) GetBase() *reference.Local {
	return p.GetLocal()
}

func (p *LazySelfRef) GetScope() reference.Scope {
	ss := p.local.SubScope()
	return ss.AsBaseOf(ss)
}
