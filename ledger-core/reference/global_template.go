package reference

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Template struct {
	base  Local
	local LocalHeader
}

const selfRefTemplate pulse.Number = 1

func NewSelfRefTemplate(pn pulse.Number, scope SelfScope) Template {
	return NewSelfRefTemplateExt(pn, scope.Scope())
}

func NewSelfRefTemplateExt(pn pulse.Number, scope Scope) Template {
	if !pn.IsSpecialOrTimePulse() {
		panic(throw.IllegalValue())
	}
	return Template{
		local: NewLocalHeader(pn, scope.SubScope()),
		base:  Local{pulseAndScope: NewLocalHeader(selfRefTemplate, scope.SuperScope())}}
}

func NewRecordRefTemplate(pn pulse.Number) Template {
	if !pn.IsSpecialOrTimePulse() {
		panic(throw.IllegalValue())
	}
	// base gets pulseNumber == 0
	return Template{local: NewLocalHeader(pn, 0)}
}

func NewRefTemplate(template Holder, pn pulse.Number) Template {
	if !pn.IsSpecialOrTimePulse() {
		panic(throw.IllegalValue())
	}

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

func (v Template) GetScope() Scope {
	return v.base.SubScope().AsBaseOf(v.local.SubScope())
}

func (v Template) isSelfRef() bool {
	return v.base.Pulse() == selfRefTemplate
}

func (v Template) CanAsRecord() bool {
	if v.local == 0 {
		return false
	}
	return v.base.pulseAndScope == 0 || v.isSelfRef()
}

func (v Template) HasBase() bool {
	return v.base.pulseAndScope != 0
}

func (v Template) LocalHeader() LocalHeader {
	if v.local == 0 {
		panic(throw.IllegalState())
	}
	return v.local
}

// WithHash panics on zero or record ref template
func (v Template) WithHash(h LocalHash) Global {
	switch {
	case v.local == 0:
		panic(throw.IllegalState())
	case v.base.pulseAndScope == 0:
		panic(throw.IllegalState())
	case v.isSelfRef():
		return Global{
			addressLocal: v.local.WithHash(h),
			addressBase:  v.local.WithSubScope(v.base.SubScope()).WithHash(h),
		}
	}
	return Global{
		addressLocal: Local{v.local, h},
		addressBase:  v.base,
	}
}

// WithHashAsSelf panics on zero or record ref template
func (v Template) WithHashAsSelf(h LocalHash) Global {
	switch {
	case v.local == 0:
		panic(throw.IllegalState())
	case v.base.pulseAndScope == 0:
		panic(throw.IllegalState())
	}

	return Global{
		addressLocal: v.local.WithHash(h),
		addressBase:  v.local.WithSubScope(v.base.SubScope()).WithHash(h),
	}
}

// WithHashAsLocal is valid for record ref or self ref template. Panics otherwise.
func (v Template) WithHashAsRecord(h LocalHash) Local {
	switch {
	case v.local == 0:
		panic(throw.IllegalState())
	case v.base.pulseAndScope == 0:
		//
	case !v.isSelfRef():
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

var _ Holder = &MutableTemplate{}

type MutableTemplate struct {
	base    Local
	local   Local
	hasHash bool
}

func (p *MutableTemplate) IsZero() bool {
	return p.local.IsZero()
}

func (p *MutableTemplate) IsEmpty() bool {
	return !p.hasHash
}

func (p *MutableTemplate) isSelfRef() bool {
	return p.base.Pulse() == selfRefTemplate
}

func (p *MutableTemplate) GetScope() Scope {
	return p.base.SubScope().AsBaseOf(p.local.SubScope())
}

func (p *MutableTemplate) GetLocal() Local {
	if !p.hasHash {
		panic(throw.IllegalState())
	}
	return p.local
}

func (p *MutableTemplate) GetBase() Local {
	switch {
	case !p.hasHash:
		panic(throw.IllegalState())
	case p.base.pulseAndScope == 0:
		return Local{}
	case !p.isSelfRef():
		return p.base
	case p.base.SubScope() == p.local.SubScope():
		return p.local
	default:
		return Local{p.local.pulseAndScope.WithSubScope(p.base.SubScope()), p.local.hash}
	}
}

// WithHash panics on zero template
func (p *MutableTemplate) WithHash(h LocalHash) Global {
	switch {
	case p.local.IsZero():
		panic(throw.IllegalState())
	case p.base.pulseAndScope == 0:
		panic(throw.IllegalState())
	case !p.isSelfRef():
		return Global{
			addressLocal: Local{p.local.pulseAndScope, h},
			addressBase:  p.base,
		}
	case p.base.SubScope() == p.local.SubScope():
		return NewSelf(Local{p.local.pulseAndScope, h})
	default:
		return Global{
			addressLocal: Local{p.local.pulseAndScope, h},
			addressBase:  Local{p.local.pulseAndScope.WithSubScope(p.base.SubScope()), h},
		}
	}
}

// WithHashAsLocal is valid for record ref or self ref template. Panics otherwise.
func (p *MutableTemplate) WithHashAsRecord(h LocalHash) Local {
	switch {
	case p.local.IsZero():
		panic(throw.IllegalState())
	case p.base.pulseAndScope == 0:
		//
	case !p.isSelfRef():
		panic(throw.IllegalState())
	}
	return Local{p.local.pulseAndScope, h}
}

func (p *MutableTemplate) CanAsRecord() bool {
	return p.base.pulseAndScope == 0 || p.isSelfRef()
}

func (p *MutableTemplate) HasBase() bool {
	return p.base.pulseAndScope != 0
}

// SetHash enables MustGlobal() and MustLocal(). Can be called multiple times to change hash.
func (p *MutableTemplate) SetHash(h LocalHash) {
	if p.local.IsZero() {
		panic(throw.IllegalState())
	}
	p.hasHash = true
	p.local.hash = h
}

func (p *MutableTemplate) HasHash() bool {
	return p.hasHash
}

func (p *MutableTemplate) MustGlobal() Global {
	switch {
	case !p.hasHash:
		panic(throw.IllegalState())
	case p.base.pulseAndScope == 0:
		if p.local.pulseAndScope == 0 {
			return Global{}
		}
		panic(throw.IllegalState())
	}
	return Global{addressLocal: p.local, addressBase: p.GetBase()}
}

func (p *MutableTemplate) MustRecord() Local {
	switch {
	case !p.hasHash:
		panic(throw.IllegalState())
	case !p.CanAsRecord():
		panic(throw.IllegalState())
	}
	return p.local
}

func (p *MutableTemplate) AsTemplate() Template {
	if p.local.IsZero() {
		panic(throw.IllegalState())
	}
	return Template{base: p.base, local: p.local.GetHeader()}
}

func (p *MutableTemplate) SetZeroValue() {
	p.base = Local{}
	p.local = Local{}
	p.hasHash = true
}
