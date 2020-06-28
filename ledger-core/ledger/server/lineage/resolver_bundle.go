// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type UnresolvedDependency struct {
	RecordRef    reference.Holder // nil for dependency on filament root
	LocalRootRef reference.Holder // nil for dependency on other lines
}

func newBundleResolver(resolver lineResolver, policyProvider RecordPolicyProviderFunc) *BundleResolver {
	if policyProvider == nil {
		panic(throw.IllegalValue())
	}
	return &BundleResolver{
		resolver: resolver,
		policyProvider: policyProvider,

		maxRecNo: resolver.getNextRecNo(),
		maxFilNo: resolver.getNextFilNo(),
	}
}

type BundleResolver struct {
	resolver       lineResolver
	policyProvider RecordPolicyProviderFunc

	maxRecNo   recordNo // can't reference a record after this point
	maxFilNo   filamentNo

	lastRecord reference.Holder
	unresolved []UnresolvedDependency
	errors     []error

	records    []resolvedRecord
}

func (p *BundleResolver) Hint(recordCount int) {
	if p.records != nil {
		panic(throw.IllegalState())
	}
	p.records = make([]resolvedRecord, 0, recordCount)
}

func (p *BundleResolver) GetUnresolvedDependencies() []UnresolvedDependency {
	return p.unresolved
}

func (p *BundleResolver) GetErrors() []error {
	return p.errors
}

func (p *BundleResolver) IsEmpty() bool {
	return len(p.records) == 0
}

func (p *BundleResolver) IsResolved() bool {
	return len(p.records) > 0 && len(p.errors) == 0 && len(p.unresolved) == 0
}

func (p *BundleResolver) Reprocess() bool {
	if p.IsEmpty() {
		return false
	}
	records := p.records
	p.records = nil
	p.errors = nil
	p.unresolved = nil

	p.Hint(len(records))
	for i := range records {
		p.Add(records[i].Record)
	}

	return p.IsResolved()
}

func (p *BundleResolver) Add(record Record) bool {
	upd := resolvedRecord{
		recordNo: p.maxRecNo + recordNo(len(p.records)),
		Record:   record,
	}

	ref := upd.RegRecord.AnticipatedRef.Get()
	p.setLastRecord(ref)
	defer p.setLastRecord(nil)

	policy := GetRecordPolicy(RecordType(upd.Excerpt.RecordType))
	if !policy.IsValid() {
		return p.addError(throw.E("unknown type"))
	}

	isResolved := false

	switch base := p.resolver.getLineBase().GetLocal(); {
	case !p.checkBase(base, "RecordRef", ref):
	case !p.checkBase(base, "RootRef", upd.Excerpt.RootRef.Get()):
	case !p.checkBase(base, "PrevRef", upd.Excerpt.PrevRef.Get()):
	case !p.checkBase(base, "RedirectRef", upd.Excerpt.RedirectRef.Get()):
	case !p.checkBase(base, "RejoinRef", upd.Excerpt.RejoinRef.Get()):
	default:
		localPN := p.resolver.getLocalPN()
		if err := policy.CheckRecordRef(base, localPN, ref); err != nil {
			return p.addRefError("RecordRef", err)
		}
//		upd.policy = policy

		refLocal := ref.GetLocal()
		switch foundNo, err := p.resolver.findCollision(refLocal, &upd.Record); {
		case err != nil:
			return p.addRefError("RecordRef", err)
		case foundNo != 0:
			if foundNo >= p.maxRecNo {
				panic(throw.IllegalState())
			}
			upd.recordNo = foundNo
			return true
		case p.findResolved(refLocal) != nil:
			return p.addRefError("RecordRef", throw.E("duplicate"))
		case !policy.IsAnyFilamentStart():
		case len(p.records) != 0:
			return p.addError(throw.E("can only be first in bundle"))
		}

		isResolved = p.resolveRecordDependencies(localPN, &upd, policy)
	}

	p.records = append(p.records, upd)
	return isResolved

	// 	prevRec := p.getRecord(upd.prev)
	// 	if prevRec.next != 0 {
	// 		return throw.E("branching detected", struct { Ref reference.Global }{ reference.Copy(ref) })
	// 	}
	// 	prevRec.next = upd.recordNo
	// }
}

func (p *BundleResolver) resolveRecordDependencies(localPN pulse.Number, upd *resolvedRecord, policy RecordPolicy) bool {
	isResolved := true
	rootRef := upd.Excerpt.RootRef.Get()

	if err := policy.CheckRootRef(rootRef, p.policyProvider, func(ref reference.Holder) (dep ResolvedDependency, _ error) {
		if ok, dp := p.getLocalFilament(ref); ok {
			return dp, nil
		}

		switch upd.filNo, dep = p.resolver.findFilament(ref); {
		case upd.filNo >= p.maxFilNo:
			return ResolvedDependency{}, throw.E("forward filament")
		case dep.IsNotAvailable():
			isResolved = false
			p.addDependency(nil, ref)
		}
		return
	}); err != nil {
		isResolved = false
		p.addRefError("RootRef", err)
	}

	if err := policy.CheckPrevRef(localPN, upd.Excerpt.PrevRef.Get(), func(ref reference.Holder) (dep ResolvedDependency, _ error) {
		filNo, recNo, isOpen := filamentNo(0), recordNo(0), false

		rd := p.findResolved(ref)
		if rd != nil {
			filNo = rd.filNo
			recNo = rd.recordNo
			dep = rd.asResolvedDependency()
			isOpen = rd.next == 0
		} else {
			filNo, recNo, isOpen, dep = p.resolver.findLocalDependency(rootRef, ref)
			if recNo == 0 {
				filNo, upd.recapNo, isOpen, dep = p.resolver.findLineDependencyWithRecap(rootRef, ref)
				if dep.IsNotAvailable() || dep.IsZero() {
					return
				}
			}
			switch {
			case recNo >= p.maxRecNo:
				return ResolvedDependency{}, throw.E("forward reference")
			case filNo >= p.maxFilNo:
				return ResolvedDependency{}, throw.E("forward filament")
			case filNo == 0:
				panic(throw.Impossible())
			}
		}

		switch refPolicy := p.policyProvider(dep.RecordType); {
		case !refPolicy.IsValid():
			panic(throw.Impossible())
		case policy.IsBranched():
			// this is the first record of a filament
			// upd.filNo = 0 => filament is created in this batch
			// PrevRef doesn't need to be "open"
			filNo = 0
		case !isOpen:
			return ResolvedDependency{}, throw.E("fork forbidden")
		case refPolicy.IsFilamentStartNotBranched():
			// this is the first record of a filament
			// upd.filNo = 0 => filament is created in this batch
			filNo = 0
			fallthrough
		default:
			if rd != nil {
				rd.next = upd.recordNo
			}
		}
		upd.filNo = filNo

		return dep, nil

	}); err != nil {
		isResolved = false
		p.addRefError("PrevRef", err)
	}

	if err := policy.CheckRejoinRef(localPN, upd.Excerpt.RejoinRef.Get(), func(ref reference.Holder) (ResolvedDependency, error) {
		// rejoin must be within the same record set
		switch rd := p.findResolved(ref); {
		case rd == nil:
		case p.policyProvider(RecordType(rd.Excerpt.RecordType)).CanBeRejoined():
			return ResolvedDependency{}, throw.E("rejoin forbidden")
		default:
			return rd.asResolvedDependency(), nil
		}
		return ResolvedDependency{}, nil

	}); err != nil {
		isResolved = false
		p.addRefError("RejoinRef", err)
	}

	if err := policy.CheckRedirectRef(upd.Excerpt.RedirectRef.Get(), func(ref reference.Holder) (ResolvedDependency, error) {
		// RedirectRef is NOT supported to records in the same batch (meaningless)
		switch recNo, dep := p.resolver.findLineDependency(rootRef, ref); {
		case dep.IsNotAvailable():
			isResolved = false
			if ref.GetBase() == rootRef.GetBase() {
				p.addDependency(ref, rootRef)
			} else {
				p.addDependency(ref, nil)
			}
			return dep, nil
		case recNo >= p.maxRecNo:
			return ResolvedDependency{}, throw.E("forward reference")
		default:
			upd.redirectToType = dep.RedirectToType
			return dep, nil
		}

	}); err != nil {
		isResolved = false
		p.addRefError("RedirectRef", err)
	}

	return isResolved
}

func (p *BundleResolver) addRefError(name string, err error) bool {
	if err == nil {
		panic(throw.IllegalValue())
	}
	return p.addError(throw.WithDetails(err, struct { FieldName string }{ name }))
}

func (p *BundleResolver) addError(err error) bool {
	if err == nil {
		panic(throw.IllegalValue())
	}
	if p.lastRecord != nil {
		err = throw.WithDetails(err, struct { Ref reference.Global }{ reference.Copy(p.lastRecord) })
	}
	p.errors = append(p.errors, err)
	return false
}

func (p *BundleResolver) checkBase(base reference.Local, fieldName string, ref reference.Holder) bool {
	if ref.IsEmpty() {
		return true
	}
	if refBase := ref.GetBase(); base != refBase {
		return p.addRefError(fieldName, throw.E("wrong base", struct{ Expected reference.Local }{base}))
	}
	return true
}

func (p *BundleResolver) setLastRecord(ref reference.Holder) {
	p.lastRecord = ref
}

func (p *BundleResolver) getLocalFilament(root reference.Holder) (bool, ResolvedDependency) {
	switch {
	case len(p.records) == 0:
	case root == nil:
	case !reference.Equal(root, p.records[0].RegRecord.AnticipatedRef.Get()):
	case !p.records[0].Excerpt.RootRef.IsEmpty():
		// not a filament
		return true, ResolvedDependency{}
	default:
		r := p.records[0]
		return true, ResolvedDependency{
			RecordType:     RecordType(r.Excerpt.RecordType),
			RootRef:        r.Excerpt.RootRef.Get(),
		}
	}
	return false, ResolvedDependency{}
}

func (p *BundleResolver) findResolved(ref reference.LocalHolder) *resolvedRecord {
	if len(p.records) == 0 {
		return nil
	}

	refLocal := ref.GetLocal()
	for i := range p.records {
		r := &p.records[i]
		l := r.RegRecord.AnticipatedRef.Get().GetLocal()
		if refLocal == l {
			return r
		}
	}

	return nil
}

func (p *BundleResolver) addDependency(ref reference.Holder, root reference.Holder) {
	if ref == nil && root == nil {
		panic(throw.IllegalValue())
	}

	for _, d := range p.unresolved {
		if reference.Equal(ref, d.RecordRef) && reference.Equal(root, d.LocalRootRef) {
			return
		}
	}

	p.unresolved = append(p.unresolved, UnresolvedDependency{ ref, root })
}

