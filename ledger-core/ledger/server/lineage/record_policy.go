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

type FieldPolicy uint16

const (
	// RedirectToNothing
	LineStart FieldPolicy = 1<<iota
	FilamentStart
	FilamentEnd
	Branched
	SideEffect
	OnlyHash
	NextPulseOnly
	Unblocked // ?
	Recap // can follow up any record (Deactivate?)
	ReasonRequired
)

type RecordPolicy struct {
	FieldPolicy
	CanFollow RecordTypeSet
	RedirectTo RecordTypeSet
}

type PolicyCheckDetails struct {
	RecordType
	LocalPN pulse.Number
	PolicyProvider RecordPolicyProviderFunc
}

func (v RecordPolicy) IsValid() bool {
	return v.FieldPolicy != 0 || !v.CanFollow.IsZero()
}

func (v RecordPolicy) CheckRecordRef(lineBase reference.LocalHolder, ref reference.LocalHolder, details PolicyCheckDetails) error {
	refLocal := ref.GetLocal()
	if ss := refLocal.SubScope(); ss != reference.SubScopeLifeline {
		return throw.E("invalid scope", struct { Actual reference.SubScope }{ ss })
	}

	switch {
	case v.FieldPolicy&LineStart == 0:
		//
	case v.FieldPolicy&Branched == 0:
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
	if ok, err := checkExact(ref, v.FieldPolicy&(LineStart|Branched) != LineStart); !ok || err != nil {
		return err
	}

	switch found, err := resolverFn(ref); {
	case err != nil:
		return err
	case found.IsZero():
		return throw.E("missing record")
	case found.RecordType == RecordNotAvailable:
	case details.PolicyProvider == nil:
	case v.FieldPolicy&LineStart != 0:
	default:
		msg := ""
		switch policy := details.PolicyProvider(found.RecordType); {
		case !policy.IsValid():
			msg = "unknown root type"
		case policy.FieldPolicy & (LineStart|FilamentStart) == 0:
			msg = "wrong root type"
		default:
			return nil
		}
		return throw.E(msg, struct { RecordType }{ found.RecordType })
	}
	return nil
}

func (v RecordPolicy) CheckPrevRef(ref reference.Holder, details PolicyCheckDetails, resolverFn PolicyResolverFunc) error {
	if ok, err := checkExact(ref, v.FieldPolicy&LineStart == 0); !ok || err != nil {
		return err
	}

	switch {
	case v.FieldPolicy&NextPulseOnly == 0:
	case ref.GetLocal().GetPulseNumber() >= details.LocalPN:
		return throw.E("must be prev pulse")
	}

	switch found, err := resolverFn(ref); {
	case err != nil:
		return err
	case found.IsZero():
		return throw.E("unknown record")
	case found.RecordType == RecordNotAvailable:
	case v.FieldPolicy&Recap != 0:
	case v.CanFollow.Has(found.RecordType):
	case found.RedirectToRef == nil:
	case v.CanFollow.Has(found.RedirectToType):
	default:
		return throw.E("wrong record sequence", struct { PrevType RecordType }{found.RecordType })
	}
	return nil
}

func (v RecordPolicy) CheckRejoinRef(ref reference.Holder, details PolicyCheckDetails, prevRecordType RecordType, resolverFn PolicyResolverFunc) error {
	isSideEffect := v.FieldPolicy&SideEffect != 0

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

	switch found, err := resolverFn(ref); {
	case err != nil:
		return err
	case found.IsZero():
		return throw.E("unknown record")
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
	if ok, err := checkExact(ref, v.FieldPolicy&ReasonRequired != 0); !ok || err != nil {
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
	return v.FieldPolicy&(LineStart|FilamentStart) != 0
}

func (v RecordPolicy) IsBranched() bool {
	return v.FieldPolicy&Branched != 0
}

func (v RecordPolicy) IsFilamentStart() bool {
	return v.FieldPolicy&FilamentStart != 0
}

func (v RecordPolicy) CanBeRejoined() bool {
	return v.FieldPolicy&(SideEffect|FilamentEnd) == FilamentEnd
}

func (v RecordPolicy) IsForkAllowed() bool {
	return v.FieldPolicy&(FilamentStart|Branched) == FilamentStart
}
