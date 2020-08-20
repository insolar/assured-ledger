// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package lineage

import (
	"math"

	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type RecordType uint32

const (
	RecordNotFound RecordType = 0
	RecordNotAvailable RecordType = math.MaxUint32
)

type PolicyFlags uint32

const (
	LineStart PolicyFlags = 1<<iota
	FilamentStart
	FilamentEnd
	BranchedStart
	MustBeBranch
	SideEffect
	OnlyHash
	NextPulseOnly
	BlockNextPulse
	Recap // can follow up any record (Deactivate?)
	ReasonRequired
	PayloadAllowed
)

type DustMode uint8

const (
	_ DustMode = iota
	DustPayload
	DustRecord
)

type RecordPolicy struct {
	PolicyFlags
	CanFollow RecordTypeSet
	RedirectTo RecordTypeSet
}

type PolicyCheckDetails struct {
	RecordType
	LocalPN pulse.Number
	PolicyProvider RecordPolicyProviderFunc
}

func (v RecordPolicy) IsValid() bool {
	return v.PolicyFlags != 0 || !v.CanFollow.IsZero()
}

func (v RecordPolicy) CheckRecordRef(lineBase reference.LocalHolder, ref reference.LocalHolder, details PolicyCheckDetails) error {
	refLocal := ref.GetLocal()
	if ss := refLocal.SubScope(); ss != reference.SubScopeLifeline {
		return throw.E("invalid scope", struct { Actual reference.SubScope }{ ss })
	}

	switch {
	case v.PolicyFlags&LineStart == 0:
		//
	case v.PolicyFlags&BranchedStart == 0:
		if refLocal.GetLocal() != lineBase.GetLocal() {
			return throw.E("must be self-ref")
		}
	}

	if pn := refLocal.GetPulseNumber(); details.LocalPN != pn {
		return throw.E("wrong pulse number", struct { Expected, Actual pulse.Number }{ details.LocalPN, pn })
	}

	return nil
}

func (v RecordPolicy) CheckRootRef(ref reference.Holder, details PolicyCheckDetails, resolverFn PolicyResolverFunc) error {
	if ok, err := checkExact(ref, v.PolicyFlags&(LineStart|BranchedStart) != LineStart); !ok || err != nil {
		return err
	}

	switch found, err := resolverFn(ref); {
	case err != nil:
		return err
	case found.IsZero():
		return throw.E("missing record")
	case found.RecordType == RecordNotAvailable:
	case details.PolicyProvider == nil:
	case v.PolicyFlags&LineStart != 0:
	default:
		msg := ""
		switch policy := details.PolicyProvider(found.RecordType); {
		case !policy.IsValid():
			msg = "unknown root type"
		case policy.PolicyFlags& (LineStart|FilamentStart) == 0:
			msg = "wrong root type"
		default:
			return nil
		}
		return throw.E(msg, struct { RecordType }{ found.RecordType })
	}
	return nil
}

func (v RecordPolicy) CheckPrevRef(rootRef, ref reference.Holder, details PolicyCheckDetails, resolverFn PolicyResolverFunc) (isFork bool, err error) {
	err = v.checkPrevRef(rootRef, ref, details, resolverFn, &isFork)
	return
}

func (v RecordPolicy) checkPrevRef(rootRef, ref reference.Holder, details PolicyCheckDetails, resolverFn PolicyResolverFunc, isFork *bool) error {
	if ok, err := checkExact(ref, v.PolicyFlags&LineStart == 0); !ok || err != nil {
		return err
	}

	switch {
	case v.PolicyFlags&NextPulseOnly == 0:
	case ref.GetLocal().GetPulseNumber() >= details.LocalPN:
		return throw.E("must be prev pulse")
	}

	found, err := resolverFn(ref)
	switch {
	case err != nil:
		return err
	case found.IsZero():
		return throw.E("unknown record")
	case found.RecordType == RecordNotAvailable:
		return nil
	case v.PolicyFlags&Recap != 0:
		if !reference.IsEmpty(found.RootRef) && !reference.Equal(rootRef, found.RootRef) {
			return throw.E("filament crossing is forbidden")
		}
		return nil
	case v.CanFollow.Has(found.RecordType):
	case found.RedirectToRef == nil:
	case v.CanFollow.Has(found.RedirectToType):
	default:
		return throw.E("wrong record sequence", struct{ PrevType RecordType }{found.RecordType})
	}

	prevPolicy := details.PolicyProvider(found.RecordType)

	switch {
	case prevPolicy.PolicyFlags&BlockNextPulse == 0:
	case ref.GetLocal().GetPulseNumber() == details.LocalPN:
	case found.RecordType == tRLineMemoryExpected:
		switch details.RecordType {
		case tRLineMemoryProvided, tRLineDeactivate:
		default:
			return throw.E("memory is expected")
		}
	default:
		panic(throw.Unsupported())
	}

	switch {
	case v.IsBranched():
		// this is the start and the first record of a branch filament
		// PrevRef doesn't need to be "open"
		// upd.filNo = 0 // => filament is created in this batch
		*isFork = true
	case reference.Equal(rootRef, ref):
		// this is not start but a first record of a filament
		if details.PolicyProvider(found.RecordType).IsForkAllowed() {
			// PrevRef doesn't need to be "open"
			// upd.filNo = 0 // => filament is created in this batch
			*isFork = true
		}
//		case v.PolicyFlags&MustBeBranch != 0:
		// TODO protection from placing ROutboundRequest inline with RLineInboundRequest
		// if reference.Equal(rootRef, found.RootRef) {
		// 	return throw.E("fork is required")
		// }
	case !reference.IsEmpty(found.RootRef) && !reference.Equal(rootRef, found.RootRef):
		return throw.E("filament crossing is forbidden")
	}

	return nil
}

func (v RecordPolicy) CheckRejoinRef(ref reference.Holder, details PolicyCheckDetails, prevRecordType RecordType, resolverFn PolicyResolverFunc) error {
	isSideEffect := v.PolicyFlags&SideEffect != 0

	if isSideEffect && details.RecordType == tRLineActivate && setDirectActivate.Has(prevRecordType) {
		// This is a special case - RLineActivate can be produced without side-effect when constructor had no outgoing calls
		isSideEffect = false
	}

	if ok, err := checkExact(ref, isSideEffect); !ok || err != nil {
		return err
	}

	if ref.GetLocal().GetPulseNumber() != details.LocalPN {
		return throw.E("must be same pulse")
	}

	found, err := resolverFn(ref)
	switch {
	case err != nil:
		return err
	case found.IsZero():
		return throw.E("unknown record")
	}

	joinPolicy := details.PolicyProvider(found.RecordType)
	if !joinPolicy.CanBeRejoined() {
		return throw.E("rejoin forbidden")
	}

	return nil
}

func (v RecordPolicy) CheckRedirectRef(ref reference.Holder, resolverFn PolicyResolverFunc) error {
	if ok, err := checkExact(ref, !v.RedirectTo.IsZero()); !ok || err != nil {
		return err
	}

	switch found, err := resolverFn(ref); {
	case err != nil:
		return err
	case found.IsZero():
		return throw.E("unknown record")
	case found.RecordType == RecordNotAvailable:
	case found.RedirectToRef != nil:
		return throw.E("redirect to redirect is forbidden")
	case v.RedirectTo.Has(found.RecordType):
	default:
		return throw.E("wrong redirect target", struct { PrevType RecordType }{found.RecordType })
	}
	return nil
}

func (v RecordPolicy) CheckReasonRef(ref reference.Holder, resolverFn PolicyResolverFunc) error {
	if ok, err := checkExact(ref, v.PolicyFlags&ReasonRequired != 0); !ok || err != nil {
		return err
	}

	switch found, err := resolverFn(ref); {
	case err != nil:
		return err
	case found.IsZero():
		return throw.E("unknown record")
	}
	return nil
}

func checkExact(ref reference.Holder, isRequired bool) (bool, error) {
	switch {
	case isRequired:
		if reference.IsEmpty(ref) {
			return false, throw.E("must be not empty")
		}
		return true, nil
	case !reference.IsEmpty(ref):
		return false, throw.E("must be empty")
	}
	return false, nil
}

func (v RecordPolicy) IsAnyFilamentStart() bool {
	return v.PolicyFlags&(LineStart|FilamentStart) != 0
}

func (v RecordPolicy) IsBranched() bool {
	return v.PolicyFlags&BranchedStart != 0
}

func (v RecordPolicy) IsFilamentStart() bool {
	return v.PolicyFlags&FilamentStart != 0
}

func (v RecordPolicy) CanBeRejoined() bool {
	return v.PolicyFlags&(SideEffect|FilamentEnd) == FilamentEnd
}

func (v RecordPolicy) IsForkAllowed() bool {
	return v.PolicyFlags&(FilamentStart|BranchedStart) == FilamentStart
}
