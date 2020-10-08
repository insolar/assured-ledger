// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rmsbox

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ rmsreg.GoGoSerializableWithText = &Reference{}

func NewReference(v reference.Holder) Reference {
	r := Reference{}
	r.Set(v)
	return r
}

func NewReferenceLazy(v ReferenceProvider) Reference {
	r := Reference{}
	r.SetLazy(v)
	return r
}

func NewReferenceLocal(v reference.LocalHolder) Reference {
	r := Reference{}
	r.SetWithoutBase(v)
	return r
}

type Reference struct {
	value reference.Holder
	lazy  ReferenceProvider
}

func (p *Reference) resolveLazy() {
	if p.lazy == nil {
		return
	}
	ref := p.lazy.TryPullReference()
	if ref.IsZero() {
		panic(throw.IllegalState())
	}
	p.lazy = nil
	p.value = ref
}

func (p *Reference) ProtoSize() int {
	p.resolveLazy()
	if p.value == nil {
		return 0
	}
	sz := reference.BinarySize(p.value)
	if sz > 0 {
		return protokit.BinaryProtoSize(sz)
	}
	return protokit.ExplicitEmptyBinaryProtoSize
}

func (p *Reference) MarshalTo(b []byte) (int, error) {
	p.resolveLazy()
	if p.value == nil {
		return 0, nil
	}
	return protokit.BinaryMarshalTo(b, true, func(b []byte) (int, error) {
		return reference.MarshalTo(p.value, b)
	})
}

func (p *Reference) MarshalToSizedBuffer(b []byte) (int, error) {
	p.resolveLazy()
	if p.value == nil {
		return 0, nil
	}

	return protokit.BinaryMarshalToSizedBuffer(b, true, func(b []byte) (int, error) {
		return reference.MarshalToSizedBuffer(p.value, b)
	})
}

func (p *Reference) Unmarshal(b []byte) error {
	p.lazy = nil
	if len(b) == 0 {
		p.value = nil
		return nil
	}

	err := protokit.BinaryUnmarshal(b, func(b []byte) (err error) {
		p.value, err = reference.UnmarshalGlobal(b)
		return err
	})
	if err != nil {
		p.value = nil
	}
	return err
}

func (p *Reference) Set(holder reference.Holder) {
	p.lazy = nil
	if reference.IsEmpty(holder) {
		p.value = nil
	} else {
		p.value = holder
	}
}

func (p *Reference) SetExact(holder reference.Holder) {
	p.lazy = nil
	p.value = holder
}

func (p *Reference) SetLazy(lazy ReferenceProvider) {
	p.lazy = lazy
	p.value = nil
}

func (p *Reference) Get() reference.Holder {
	p.resolveLazy()
	return p.value
}

func (p *Reference) GetValue() reference.Global {
	return reference.Copy(p.Get())
}

// SetWithoutBase ignores base part and sets only local one.
func (p *Reference) SetWithoutBase(holder reference.LocalHolder) {
	p.lazy = nil
	if holder == nil {
		p.value = nil
		return
	}
	p.value = reference.New(reference.Local{}, holder.GetLocal())
}

// GetWithoutBase returns only local part and PANICS when base part is present.
func (p *Reference) GetWithoutBase() reference.LocalHolder {
	p.resolveLazy()
	if p.value == nil {
		return nil
	}
	local := p.value.GetLocal()
	base := p.value.GetBase()
	if base.IsEmpty() || local == base {
		return local
	}
	panic(throw.IllegalState())
}

// GetValueWithoutBase returns only local part and PANICS when base part is present.
func (p *Reference) GetValueWithoutBase() reference.Local {
	return reference.CopyLocal(p.GetWithoutBase())
}

// GetPulseOfLocal returns pulse number of local portion of reference. Returns zero when reference is not set or empty.
func (p *Reference) GetPulseOfLocal() pulse.Number {
	p.resolveLazy()
	if p.value == nil {
		return 0
	}
	return reference.PulseNumberOf(p.value)
}

// GetPulseOfBase returns pulse number of base portion of reference. Returns zero when reference is not set, is empty or has no base part.
func (p *Reference) GetPulseOfBase() pulse.Number {
	p.resolveLazy()
	if p.value == nil {
		return 0
	}
	return reference.PulseNumberOf(p.value.GetBase())
}

func (p *Reference) Equal(o *Reference) bool {
	switch {
	case p == o:
		return true
	case p == nil || o == nil:
		return false
	case p.value == o.value:
		return true
	}
	return reference.Equal(p.value, o.value)
}

func (p *Reference) MarshalText() ([]byte, error) {
	switch {
	case p.value != nil:
		return []byte(reference.Encode(p.value)), nil
	case p.lazy != nil:
		return []byte("lazy-ref"), nil
	default:
		return nil, nil
	}
}

func (p *Reference) IsZero() bool {
	return p.value == nil
}

func (p *Reference) IsEmpty() bool {
	return p.value == nil || p.value.IsEmpty()
}

func (p *Reference) NormEmpty() Reference {
	p.resolveLazy()
	if p.value != nil && p.value.IsEmpty() {
		return Reference{}
	}
	return *p
}

