// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

import (
	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

type Template struct {
	base  Local
	local LocalHeader
}

func NewSelfRefTemplate(pn pulse.Number, scope SelfScope) Template {
	if !pn.IsSpecialOrTimePulse() {
		panic(throw.IllegalValue())
	}
	h := NewLocalHeader(pn, scope.SubScope())
	return Template{local: h, base: Local{pulseAndScope: h}}
}

func NewRecordRefTemplate(pn pulse.Number) Template {
	if !pn.IsSpecialOrTimePulse() {
		panic(throw.IllegalValue())
	}
	// base gets pulseNumber == 0
	return Template{local: NewLocalHeader(pn, 0)}
}

func NewRefTemplate(template Holder, pn pulse.Number) Template {
	return NewRefTemplateExt(template.GetBase(), template.GetLocal().GetHeader().WithPulse(pn))
}

func NewRefTemplateExt(base Local, local LocalHeader) Template {
	switch {
	case !local.Pulse().IsSpecialOrTimePulse():
		panic(throw.IllegalValue())
	case !base.Pulse().IsSpecialOrTimePulse():
		panic(throw.IllegalValue())
	}
	return Template{local: local, base: base}
}

func (v Template) IsZero() bool {
	return v.local == 0
}

// WithHash panics on zero or record ref template
func (v Template) WithHash(h LocalHash) Global {
	switch {
	case v.local == 0:
		panic(throw.IllegalState())
	case v.base.pulseAndScope == 0:
		panic(throw.IllegalState())
	}
	return Global{
		addressLocal: Local{v.local, h},
		addressBase:  Local{v.base.pulseAndScope, h},
	}
}

// WithHashAsLocal is valid for record ref or self ref template. Panics otherwise.
func (v Template) WithHashAsLocal(h LocalHash) Local {
	switch {
	case v.local == 0:
		panic(throw.IllegalState())
	case v.base.pulseAndScope == 0:
		//
	case v.base.pulseAndScope != v.local:
		panic(throw.IllegalState())
	}
	return Local{v.local, h}
}

func (v Template) AsMutable() MutableTemplate {
	if v.IsZero() {
		panic(throw.IllegalValue())
	}
	return MutableTemplate{base: v.base, local: Local{pulseAndScope: v.local}}
}

/**************************/

type MutableTemplate struct {
	base    Local
	local   Local
	hasHash bool
}

func (p *MutableTemplate) IsZero() bool {
	return p.local.IsZero()
}

// WithHash panics on zero or record ref template
func (p *MutableTemplate) WithHash(h LocalHash) Global {
	switch {
	case p.local.IsZero():
		panic(throw.IllegalState())
	case p.base.pulseAndScope == 0:
		panic(throw.IllegalState())
	}
	return Global{
		addressLocal: Local{p.local.pulseAndScope, h},
		addressBase:  Local{p.base.pulseAndScope, h},
	}
}

// WithHashAsLocal is valid for record ref or self ref template. Panics otherwise.
func (p *MutableTemplate) WithHashAsLocal(h LocalHash) Local {
	switch {
	case p.local.IsZero():
		panic(throw.IllegalState())
	case p.base.pulseAndScope == 0:
		//
	case p.base.pulseAndScope != p.local.pulseAndScope:
		panic(throw.IllegalState())
	}
	return Local{p.local.pulseAndScope, h}
}

// SetHash enables MustGlobal() and MustLocal(). Can be called multiple times to change hash.
func (p *MutableTemplate) SetHash(h LocalHash) {
	if p.local.IsZero() {
		panic(throw.IllegalState())
	}
	p.hasHash = true
	p.local.hash = h
	if p.base.pulseAndScope != 0 {
		p.base.hash = h
	}
}

func (p *MutableTemplate) MustGlobal() Global {
	switch {
	case !p.hasHash:
		panic(throw.IllegalState())
	case p.base.pulseAndScope == 0:
		panic(throw.IllegalState())
	}
	return Global{addressLocal: p.local, addressBase: p.base}
}

func (p *MutableTemplate) MustLocal() Local {
	switch {
	case !p.hasHash:
		panic(throw.IllegalState())
	case p.base.pulseAndScope == 0:
		//
	case p.base.pulseAndScope != p.local.pulseAndScope:
		panic(throw.IllegalState())
	}
	return p.local
}
