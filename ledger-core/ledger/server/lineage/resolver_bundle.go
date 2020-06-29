// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
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
	hasBranch  bool

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
	p.hasBranch = false

	// NB! required filament(s) can be retrieved from prev drops
	p.maxFilNo = p.resolver.getNextFilNo()

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

	recType := RecordType(upd.Excerpt.RecordType)
	policy := GetRecordPolicy(recType)
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
		details := PolicyCheckDetails{
			RecordType:     recType,
			LocalPN:        p.resolver.getLocalPN(),
			PolicyProvider: p.policyProvider,
		}

		if err := policy.CheckRecordRef(base, ref, details); err != nil {
			return p.addRefError("RecordRef", err)
		}

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
		case policy.IsAnyFilamentStart():
			if len(p.records) == 0 {
				if p.maxRecNo > 1 {
					p.hasBranch = true
				}
				break
			}

			switch {
			case p.hasBranch:
				return p.addError(throw.E("can only be one branch in bundle"))
			case base.GetPulseNumber() == details.LocalPN:
			case reference.PulseNumberOf(upd.Excerpt.ReasonRef.Get()) == details.LocalPN:
			default:
				return p.addError(throw.E("can only be first in bundle"))
			}

			p.hasBranch = true
		}

		isResolved = p.resolveRecordDependencies(&upd, policy, details)
	}

	p.records = append(p.records, upd)
	return isResolved
}

func (p *BundleResolver) resolveRecordDependencies(upd *resolvedRecord, policy RecordPolicy, details PolicyCheckDetails) bool {
	isResolved := true
	rootRef := upd.Excerpt.RootRef.Get()

	if err := policy.CheckRootRef(rootRef, details, func(ref reference.Holder) (dep ResolvedDependency, _ error) {
		if ok, filNo, dp := p.getLocalFilament(ref); ok {
			if dp.IsZero() {
				return ResolvedDependency{}, throw.E("not a root")
			}
			upd.filNo = filNo
			return dp, nil
		}

		switch upd.filNo, dep = p.resolver.findFilament(ref); {
		case upd.filNo >= p.maxFilNo:
			return ResolvedDependency{}, throw.E("forward filament")
		case dep.IsNotAvailable():
			isResolved = false
			p.addDependency(ref, nil) // filament roots only
		}
		return
	}); err != nil {
		isResolved = false
		p.addRefError("RootRef", err)
	}

	prevRecordType := RecordType(0)

	if err := policy.CheckPrevRef(upd.Excerpt.PrevRef.Get(), details, func(ref reference.Holder) (dep ResolvedDependency, _ error) {

		if rd := p.findResolved(ref); rd != nil {
			upd.prev = rd.recordNo
			dep = rd.asResolvedDependency()
			prevRecordType = dep.RecordType

			switch {
			case policy.IsBranched():
				// this is the start and the first record of a branch filament
				// PrevRef doesn't need to be "open"
				// upd.filNo = 0 // => filament is created in this batch
				return dep, nil
			case !reference.Equal(rootRef, ref):
				// this is neither start nor first record of a filament
			case p.policyProvider(dep.RecordType).IsForkAllowed():
				// PrevRef doesn't need to be "open"
				// upd.filNo = 0 // => filament is created in this batch
				return dep, nil
			}

			if rd.next != 0 {
				return ResolvedDependency{}, throw.E("fork forbidden")
			}
			upd.filNo = rd.filNo
			rd.next = upd.recordNo
			return dep, nil
		}

		// There are 3 cases here
		// 1. Branched is a first filament record and doesn't need an open prev
		// 2. When rootRef == ref, then it has to be a non-Branched FilamentStart
		// 3. otherwise - it has to be an open prev record

		isBranchedStart := !policy.IsBranched()
		upd.filNo, upd.prev, dep = p.resolver.findLocalDependency(rootRef, ref, !isBranchedStart)
		switch {
		case dep.IsZero():
			upd.prev = 0 // not local
			upd.filNo, dep, upd.recapNo = p.resolver.findLineDependency(rootRef, ref, !isBranchedStart)
			switch {
			case dep.IsZero():
				return
			case dep.IsNotAvailable():
				isResolved = false
				p.addDependency(rootRef, ref)
				return
			case upd.recapNo == 0:
				panic(throw.Impossible())
			}
		case dep.IsNotAvailable():
			panic(throw.Impossible())
		case upd.prev == 0 && upd.filNo != 0:
			panic(throw.Impossible())
		}

		switch {
		case upd.prev >= p.maxRecNo:
			return ResolvedDependency{}, throw.E("forward reference")
		case upd.filNo >= p.maxFilNo:
			return ResolvedDependency{}, throw.E("forward filament")
		case upd.filNo == 0:
			return ResolvedDependency{}, throw.E("fork forbidden")
		}

		prevRecordType = dep.RecordType
		return dep, nil

	}); err != nil {
		isResolved = false
		p.addRefError("PrevRef", err)
	}

	if err := policy.CheckRejoinRef(upd.Excerpt.RejoinRef.Get(), details, prevRecordType, func(ref reference.Holder) (ResolvedDependency, error) {
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

		filNo, recNo, dep := p.resolver.findLocalDependency(rootRef, ref, false)
		switch {
		case dep.IsZero():
			dep = p.resolver.findLineAnyDependency(rootRef, ref)
			switch {
			case dep.IsZero():
				return dep, nil
			case dep.IsNotAvailable():
				isResolved = false
				if ref.GetBase() == rootRef.GetBase() {
					p.addDependency(rootRef, ref)
				} else {
					p.addDependency(nil, ref)
				}
			}
			return dep, nil
		case dep.IsNotAvailable():
			panic(throw.Impossible())
		case recNo == 0 && filNo != 0:
			panic(throw.Impossible())
		}

		switch {
		case recNo >= p.maxRecNo:
			return ResolvedDependency{}, throw.E("forward reference")
		case filNo >= p.maxFilNo:
			return ResolvedDependency{}, throw.E("forward filament")
		case filNo == 0:
			return ResolvedDependency{}, throw.E("fork forbidden")
		}

		upd.redirectToType = dep.RedirectToType
		return dep, nil

	}); err != nil {
		isResolved = false
		p.addRefError("RedirectRef", err)
	}

	if err := policy.CheckReasonRef(upd.Excerpt.ReasonRef.Get(), func(ref reference.Holder) (ResolvedDependency, error) {

		if rd := p.findResolved(ref); rd != nil {
			return rd.asResolvedDependency(), nil
		}

		filNo, recNo, dep := p.resolver.findLocalDependency(rootRef, ref, false)
		switch {
		case dep.IsZero():
			dep = p.resolver.findLineAnyDependency(rootRef, ref)
			switch {
			case dep.IsZero():
				return dep, nil
			case dep.IsNotAvailable():
				isResolved = false
				if ref.GetBase() == rootRef.GetBase() {
					p.addDependency(rootRef, ref)
				} else {
					p.addDependency(nil, ref)
				}
			}
			return dep, nil
		case dep.IsNotAvailable():
			panic(throw.Impossible())
		case recNo == 0 && filNo != 0:
			panic(throw.Impossible())
		}

		switch {
		case recNo >= p.maxRecNo:
			return ResolvedDependency{}, throw.E("forward reference")
		case filNo >= p.maxFilNo:
			return ResolvedDependency{}, throw.E("forward filament")
		case filNo == 0:
			return ResolvedDependency{}, throw.E("fork forbidden")
		}

		return dep, nil

	}); err != nil {
		isResolved = false
		p.addRefError("ReasonRef", err)
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
	if ref == nil || ref.IsEmpty() {
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

func (p *BundleResolver) getLocalFilament(root reference.Holder) (bool, filamentNo, ResolvedDependency) {
	switch {
	case len(p.records) == 0:
	case root == nil:
	default:
		rd := p.findResolved(root)
		if rd == nil {
			break
		}
		rootRef := rd.Excerpt.RootRef.Get()
		filNo := filamentNo(0) // special case - a new branch filament
		if rootRef == nil {
			filNo = 1
		}

		return true, filNo, ResolvedDependency{
			RecordType:     RecordType(rd.Excerpt.RecordType),
			RootRef:        rootRef,
		}
	}
	return false, 0, ResolvedDependency{}
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

func (p *BundleResolver) addDependency(root reference.Holder, ref reference.Holder) {
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

