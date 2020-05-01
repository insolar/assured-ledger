// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import "github.com/insolar/assured-ledger/ledger-core/v2/reference"

type Reference struct {
	lazy ReferenceProvider
}

func (p *Reference) Set(holder reference.Holder) {

}

func (p *Reference) SetLocal(holder reference.LocalHolder) {

}

func (p *Reference) Get() reference.Holder {

}

func (p *Reference) GetLocal() reference.LocalHolder {

}

func (p *Reference) IsZero() bool {
	return p.value == nil
}

type ReferenceProvider interface {
	reference.Holder
}
