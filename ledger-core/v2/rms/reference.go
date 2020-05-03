// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

var _ GoGoSerializable = &Reference{}

type Reference struct {
	value reference.Holder
}

func (p *Reference) ProtoSize() int {
	return reference.ProtoSize(p.value)
}

func (p *Reference) MarshalTo(b []byte) (int, error) {
	return reference.MarshalTo(p.value, b)
}

func (p *Reference) MarshalToSizedBuffer(b []byte) (int, error) {
	return reference.MarshalToSizedBuffer(p.value, b)
}

func (p *Reference) Unmarshal(b []byte) (err error) {
	p.value, err = reference.UnmarshalGlobal(b)
	if err != nil {
		p.value = nil
	}
	return err
}

func (p *Reference) Set(holder reference.Holder) {
	p.value = holder
}

func (p *Reference) Get() reference.Holder {
	return p.value
}

func (p *Reference) GetGlobal() reference.Holder {
	return reference.Copy(p.value)
}

func (p *Reference) SetLocal(holder reference.LocalHolder) {
	if holder == nil {
		p.value = nil
		return
	}
	p.value = reference.New(reference.Local{}, holder.GetLocal())
}

func (p *Reference) GetLocal() reference.LocalHolder {
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
