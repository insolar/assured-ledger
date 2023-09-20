package rmsbox

import (
	"github.com/gogo/protobuf/proto"

	"github.com/insolar/assured-ledger/ledger-core/insproto"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/cryptkit"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type Referencable interface {
	InitFieldMap(reset bool) *insproto.FieldMap
}

func InitReferenceFactory(r Referencable, digester cryptkit.DataDigester, template reference.Template) {
	if template.IsZero() {
		panic(throw.IllegalValue())
	}
	if fm := r.InitFieldMap(false); fm.Callback != nil {
		panic(throw.IllegalState())
	}
	_setReferenceDispenser(r, digester, template)
}

func SetReferenceFactoryCanPull(r Referencable, canPull bool) {
	rd := _getReferenceDispenser(r)
	rd.canPull = canPull
}

func ForceReferenceOf(r Referencable, digester cryptkit.DataDigester, template reference.Template) reference.Global {
	switch {
	case digester == nil:
		panic(throw.IllegalValue())
	case template.IsZero():
		panic(throw.IllegalValue())
	}
	fm := r.InitFieldMap(false)

	if rd, ok := fm.Callback.(*referenceDispenser); ok && rd.isCompatibleDigester(digester) && rd.canPull {
		rp := rd.createRefProvider(template)
		if ref := rp.TryPullReference(); !ref.IsZero() {
			return ref
		}
		panic(throw.Impossible())
	}

	prevCallback := fm.Callback
	fm.Callback = nil
	defer func() {
		fm.Callback = prevCallback
	}()

	rd := &referenceDispenser{referencable: r, template: template, canPull: true}
	rd.setDigester(digester)
	fm.Callback = rd

	rp := rd.createRefProvider(template)
	if ref := rp.TryPullReference(); !ref.IsZero() {
		return ref
	}
	panic(throw.Impossible())
}

func DefaultLazyReferenceTo(r Referencable) ReferenceProvider {
	rd := _getReferenceDispenser(r)
	if rd.template.IsZero() {
		panic(throw.IllegalState())
	}
	return rd.createRefProvider(rd.template)
}

func LazyReferenceTo(r Referencable, template reference.Template) ReferenceProvider {
	return _getReferenceDispenser(r).createRefProvider(template)
}

// ReferenceToWithDigester does NOT set the template as default
func LazyReferenceToWithDigester(r Referencable, digester cryptkit.DataDigester, template reference.Template) ReferenceProvider {
	return _setReferenceDispenser(r, digester, reference.Template{}).createRefProvider(template)
}

func LazyDigestOf(r Referencable, digester cryptkit.DataDigester) DigestProvider {
	rd := _setReferenceDispenser(r, digester, reference.Template{})
	return &rd.digestProvider
}

func DefaultLazyDigestOf(r Referencable) DigestProvider {
	rd := _getReferenceDispenser(r)
	return &rd.digestProvider
}

func _getReferenceDispenser(r Referencable) *referenceDispenser {
	if fm := r.InitFieldMap(false); fm != nil && fm.Callback != nil {
		return fm.Callback.(*referenceDispenser)
	}
	panic(throw.IllegalState())
}

func _setReferenceDispenser(r Referencable, digester cryptkit.DataDigester, template reference.Template) *referenceDispenser {
	if digester == nil {
		panic(throw.IllegalValue())
	}
	fm := r.InitFieldMap(false)

	var rd *referenceDispenser
	if fm.Callback != nil {
		rd = fm.Callback.(*referenceDispenser)
		rd.checkDigester(digester)
	} else {
		rd = &referenceDispenser{referencable: r, template: template}
		rd.setDigester(digester)
		fm.Callback = rd
	}
	return rd
}

type referenceDispenser struct {
	referencable   Referencable
	template       reference.Template
	digestProvider digestProvider
	canPull        bool
}

func (p *referenceDispenser) OnMessage(fieldMap *insproto.FieldMap) {
	defer fieldMap.UnsetMap()
	if p.digestProvider.isReady() {
		return
	}
	p.digestProvider.calcDigest(func(digester cryptkit.DataDigester) cryptkit.Digest {
		return digester.DigestBytes(fieldMap.Message)
	}, nil)
}

func (p *referenceDispenser) setDigester(digester cryptkit.DataDigester) {
	p.digestProvider.setDigester(digester, nil)
}

func (p *referenceDispenser) isCompatibleDigester(digester cryptkit.DataDigester) bool {
	return p.digestProvider.GetDigestMethod() == digester.GetDigestMethod()
}

func (p *referenceDispenser) checkDigester(digester cryptkit.DataDigester) {
	if !p.isCompatibleDigester(digester) {
		panic(throw.IllegalValue())
	}
}

func (p *referenceDispenser) createRefProvider(template reference.Template) ReferenceProvider {
	return &refProvider{&p.digestProvider, template.AsMutable(), p.pull}
}

func (p *referenceDispenser) pull() {
	if !p.canPull || p.digestProvider.isReady() {
		return
	}

	var err error
	switch fm := p.referencable.InitFieldMap(false); fm.Callback {
	case nil:
		fm.Callback = p
	case p:
		//
	default:
		panic(throw.IllegalState())
	}

	switch rr := p.referencable.(type) {
	case interface{ Marshal() ([]byte, error) }:
		_, err = rr.Marshal()
	case interface {
		proto.ProtoSizer
		MarshalTo([]byte) (int, error)
	}:
		b := make([]byte, rr.ProtoSize())
		_, err = rr.MarshalTo(b)
	default:
		panic(throw.NotImplemented())
	}
	if err != nil {
		panic(throw.WithStackTop(err))
	}

	if !p.digestProvider.isReady() {
		panic(throw.IllegalState())
	}
}
