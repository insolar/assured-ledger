// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

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

	maxRecNo recordNo // can't reference a record after this point
	maxFilNo filamentNo

	resolveResults
}

//nolint:structcheck
type resolveResults struct {
	lastRecord  reference.Holder
	unresolved  []UnresolvedDependency
	errors      []error
	branchStart recordNo
	branchHead  recordNo
	isLineStart bool
	isResolved  bool

	records    []resolvedRecord
}

func (p *BundleResolver) Hint(recordCount int) {
	if p.records != nil {
		panic(throw.IllegalState())
	}
	p.records = make([]resolvedRecord, 0, recordCount)
}

func (p *BundleResolver) GetResolved() ResolvedBundle {
	if p.IsResolved() {
		return ResolvedBundle{ p.records }
	}
	panic(throw.IllegalState())
}

func (p *BundleResolver) GetUnresolvedDependencies() []UnresolvedDependency {
	return p.unresolved
}

func (p *BundleResolver) HasErrors() bool {
	return len(p.errors) > 0
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
	p.resolveResults = resolveResults{}

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

	ref := upd.GetRecordRef()
	p.setLastRecord(ref)
	defer p.setLastRecord(nil)
	p.isResolved = true

	recType := RecordType(upd.Excerpt.RecordType)
	policy := GetRecordPolicy(recType)
	if !policy.IsValid() {
		return p.addError(throw.E("unknown type"))
	}

	payloads := record.AsBasicRecord().GetRecordPayloads()
	if !payloads.IsEmpty() && policy.PolicyFlags&PayloadAllowed == 0 {
		return p.addError(throw.E("payload forbidden"))
	}

	switch base := p.resolver.getLineBase().GetLocal(); {
	case !p.checkBase(base, "RecordRef", ref):
	case !p.checkBase(base, "RootRef", upd.Excerpt.RootRef.Get()):
	case !p.checkBase(base, "PrevRef", upd.Excerpt.PrevRef.Get()):
	case !p.checkBase(base, "RedirectRef", upd.Excerpt.RedirectRef.Get()):
	case !p.checkBase(base, "RejoinRef", upd.Excerpt.RejoinRef.Get()):
	case !p.checkGenericRef("ReasonRef", upd.Excerpt.ReasonRef.Get()):
	default:
		details := PolicyCheckDetails{
			RecordType:     recType,
			LocalPN:        p.resolver.getLocalPN(),
			PolicyProvider: p.policyProvider,
		}

		if err := policy.CheckRecordRef(base, ref, details); err != nil {
			return p.addRefError("RecordRef", err)
		}

		switch hasCopy, err := p.checkCollision(&upd, ref); {
		case err != nil:
			return p.addRefError("RecordRef", err)
		case hasCopy:
			return true
		case policy.IsAnyFilamentStart():
			if len(p.records) == 0 {
				if p.maxRecNo > 1 {
					p.branchStart = upd.recordNo
				} else {
					p.isLineStart = true
					upd.filNo = 1
				}
				break
			}

			switch {
			case p.branchStart != 0:
				return p.addError(throw.E("can only be one branch in bundle"))
			case base.GetPulseNumber() == details.LocalPN:
			case reference.PulseNumberOf(upd.Excerpt.ReasonRef.Get()) == details.LocalPN:
			default:
				return p.addError(throw.E("can only be first in bundle"))
			}

			p.branchStart = upd.recordNo
		}
		if policy.PolicyFlags&FilamentEnd != 0 {
			upd.next = deadFilament
		}

		p.resolveRecordDependencies(&upd, policy, details)
	}

	p.records = append(p.records, upd)
	return p.isResolved
}

func (p *BundleResolver) checkCollision(upd *resolvedRecord, ref reference.Holder) (bool, error) {
	refLocal := ref.GetLocal()
	switch foundNo, err := p.resolver.findCollision(refLocal, &upd.Record); {
	case err != nil:
		return false, err
	case foundNo != 0:
		if foundNo >= p.maxRecNo {
			panic(throw.IllegalState())
		}
		upd.recordNo = foundNo
		return true, nil
	case p.findResolved(refLocal) != nil:
		return false, throw.E("duplicate")
	}
	return false, nil
}

func (p *BundleResolver) resolveRecordDependencies(upd *resolvedRecord, policy RecordPolicy, details PolicyCheckDetails) {
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
			p.addDependency(ref, nil) // filament roots only
		}
		return
	}); err != nil {
		p.addRefError("RootRef", err)
	}

	prevRecordType := p.resolvePrevRef(upd, policy, details)

	if err := policy.CheckRejoinRef(upd.Excerpt.RejoinRef.Get(), details, prevRecordType, func(ref reference.Holder) (ResolvedDependency, error) {
		// rejoin must be within the same record set
		if rd := p.findResolved(ref); rd != nil {
			return rd.asResolvedDependency(), nil
		}
		return ResolvedDependency{}, nil

	}); err != nil {
		p.addRefError("RejoinRef", err)
	}

	if err := policy.CheckRedirectRef(upd.Excerpt.RedirectRef.Get(), func(ref reference.Holder) (ResolvedDependency, error) {
		// RedirectRef is NOT supported to records in the same batch (meaningless)
		dep, err := p.resolveSupplementaryRef(rootRef, ref)
		if err != nil {
			return ResolvedDependency{}, err
		}

		upd.redirectToType = dep.RedirectToType
		return dep, nil

	}); err != nil {
		p.addRefError("RedirectRef", err)
	}

	// TODO re-enable
	// if err := policy.CheckReasonRef(upd.Excerpt.ReasonRef.Get(), func(ref reference.Holder) (ResolvedDependency, error) {
	// 	if rd := p.findResolved(ref); rd != nil {
	// 		return rd.asResolvedDependency(), nil
	// 	}
	//
	// 	return p.resolveSupplementaryRef(rootRef, ref)
	//
	// }); err != nil {
	// 	p.addRefError("ReasonRef", err)
	// }
}

func (p *BundleResolver) resolvePrevRef(upd *resolvedRecord, policy RecordPolicy, details PolicyCheckDetails) (prevRecordType RecordType) {
	var prevRecord *resolvedRecord
	rootRef := upd.Excerpt.RootRef.Get()

	isFork, err := policy.CheckPrevRef(rootRef, upd.Excerpt.PrevRef.Get(), details, func(ref reference.Holder) (dep ResolvedDependency, _ error) {

		if rd := p.findResolved(ref); rd != nil {
			prevRecord = rd

			upd.prev = rd.recordNo
			upd.filNo = rd.filNo
			dep = rd.asResolvedDependency()
			prevRecordType = dep.RecordType
			return dep, nil
		}

		// There are 3 cases here
		// 1. Branched is a first filament record and doesn't need an open prev
		// 2. When rootRef == ref, then it has to be a non-Branched FilamentStart
		// 3. otherwise - it has to be an open prev record

		mustBeOpen := !policy.IsBranched()
		upd.filNo, upd.prev, dep, upd.recapNo = p.resolver.findChainedDependency(rootRef, ref, mustBeOpen)
		switch {
		case dep.IsZero():
			return
		case dep.IsNotAvailable():
			if mustBeOpen {
				p.addDependency(rootRef, ref)
			} else {
				p.addDependency(nil, ref)
			}

			return
		case upd.prev == 0 && upd.filNo != 0:
			panic(throw.Impossible())
		}

		if err := p.checkOrdering(upd.prev, upd.filNo); err != nil {
			return ResolvedDependency{}, err
		}

		prevRecordType = dep.RecordType
		return dep, nil

	})

	switch {
	case err != nil:
		p.addRefError("PrevRef", err)
	case isFork:
		if p.branchHead != 0 {
			p.addRefError("PrevRef", throw.E("can only be one branch in bundle"))
		}
		switch upd.recordNo - p.branchStart {
		case 0, 1:
			p.branchHead = upd.recordNo
		default:
			p.addRefError("PrevRef", throw.E("misplaced branch fork"))
		}
		upd.filNo = 0
	case prevRecord != nil:
		if prevRecord.next != 0 {
			p.addRefError("PrevRef", throw.E("fork forbidden"))
		}
		prevRecord.next = upd.recordNo
		upd.filNo = prevRecord.filNo
	}

	return prevRecordType
}

func (p *BundleResolver) resolveSupplementaryRef(rootRef, ref reference.Holder) (ResolvedDependency, error) {
	if reference.IsEmpty(rootRef) || rootRef.GetBase() != ref.GetBase() {
		switch dep, err := p.resolver.findOtherDependency(ref); {
		case err != nil:
			return ResolvedDependency{}, nil
		case dep.IsNotAvailable():
			p.addDependency(nil, ref)
			fallthrough
		default:
			return dep, nil
		}
	}

	filNo, recNo, dep := p.resolver.findLocalDependency(rootRef, ref)
	switch {
	case dep.IsZero():
		var err error
		switch dep, err = p.resolver.findLineDependency(rootRef, ref); {
		case err != nil:
			return ResolvedDependency{}, err
		case dep.IsZero():
			return dep, nil
		case dep.IsNotAvailable():
			p.addDependency(nil, ref)
			fallthrough
		default:
			return dep, nil
		}
	case dep.IsNotAvailable():
		panic(throw.Impossible())
	case recNo == 0 && filNo != 0:
		panic(throw.Impossible())
	}

	if err := p.checkOrdering(recNo, filNo); err != nil {
		return ResolvedDependency{}, err
	}

	return dep, nil
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
	p.isResolved = false
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

func (p *BundleResolver) checkGenericRef(fieldName string, ref reference.Holder) bool {
	if ref == nil || ref.IsEmpty() {
		return true
	}
	if refBase := ref.GetBase(); refBase.IsEmpty() {
		return p.addRefError(fieldName, throw.E("empty base", struct{ Ref reference.Global }{reference.Copy(ref) }))
	}
	return true
}

func (p *BundleResolver) setLastRecord(ref reference.Holder) {
	p.lastRecord = ref
}

func (p *BundleResolver) getLocalFilament(root reference.LocalHolder) (bool, filamentNo, ResolvedDependency) {
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
		l := r.GetRecordRef().GetLocal()
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

	p.isResolved = false

	for _, d := range p.unresolved {
		if reference.Equal(ref, d.RecordRef) && reference.Equal(root, d.RecapRootRef) {
			return
		}
	}

	p.unresolved = append(p.unresolved, UnresolvedDependency{ ref, root })
}

func (p *BundleResolver) checkOrdering(recNo recordNo, filNo filamentNo) error {
	switch {
	case recNo >= p.maxRecNo:
		return throw.E("forward reference")
	case filNo >= p.maxFilNo:
		return throw.E("forward filament")
	case filNo == 0:
		return throw.E("fork forbidden")
	}
	return nil
}
