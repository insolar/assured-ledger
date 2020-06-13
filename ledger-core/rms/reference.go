// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/protokit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

var _ GoGoSerializable = &Reference{}

type Reference struct {
	value reference.Holder
	lazy  ReferenceProvider
}

func (p *Reference) ensure() {
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
	p.ensure()
	return protokit.BinaryProtoSize(reference.BinarySize(p.value))
}

func (p *Reference) MarshalTo(b []byte) (int, error) {
	return protokit.BinaryMarshalTo(b, func(b []byte) (int, error) {
		p.ensure()
		switch n := reference.BinarySize(p.value); {
		case n == 0:
			return 0, nil
		case n > len(b):
			return 0, io.ErrShortBuffer
		}
		return reference.MarshalTo(p.value, b)
	})
}

func (p *Reference) MarshalToSizedBuffer(b []byte) (int, error) {
	return protokit.BinaryMarshalToSizedBuffer(b, func(b []byte) (int, error) {
		p.ensure()
		return reference.MarshalToSizedBuffer(p.value, b)
	})
}

func (p *Reference) Unmarshal(b []byte) error {
	p.lazy = nil
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
	p.value = holder
}

func (p *Reference) SetLazy(lazy ReferenceProvider) {
	p.lazy = lazy
	p.value = nil
}

func (p *Reference) Get() reference.Holder {
	p.ensure()
	return p.value
}

func (p *Reference) GetGlobal() reference.Global {
	return reference.Copy(p.Get())
}

func (p *Reference) SetLocal(holder reference.LocalHolder) {
	p.lazy = nil
	if holder == nil {
		p.value = nil
		return
	}
	p.value = reference.New(reference.Local{}, holder.GetLocal())
}

func (p *Reference) GetLocal() reference.LocalHolder {
	p.ensure()
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

