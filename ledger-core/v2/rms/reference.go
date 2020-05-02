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
	lazy  ReferenceProvider
	value reference.Global
}

func (p *Reference) ensure() {

}

func (p *Reference) ProtoSize() int {
	p.ensure()
	return reference.ProtoSize(p.value)
}

func (p *Reference) MarshalTo(b []byte) (int, error) {
	return reference.MarshalTo(p.value, b)
}

func (p *Reference) MarshalToSizedBuffer(b []byte) (int, error) {
	panic("implement me")
}

func (p *Reference) Unmarshal(b []byte) error {
	panic("implement me")
}

func (p *Reference) Set(holder reference.Holder) {

}

func (p *Reference) SetLocal(holder reference.LocalHolder) {

}

func (p *Reference) Get() reference.Holder {
	panic(throw.NotImplemented())
}

func (p *Reference) GetLocal() reference.LocalHolder {
	panic(throw.NotImplemented())
}

func (p *Reference) IsZero() bool {
	panic(throw.NotImplemented())
}

type ReferenceProvider interface {
	reference.Holder
}
