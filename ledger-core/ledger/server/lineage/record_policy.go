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
	// EmptyRoot
	// EmptyPrev
	// AllowEmptyReason
	// AllowRedirect
	// RequireRedirect
	LineStart FieldPolicy = 1<<iota
	FilamentStart
	FilamentEnd
	Branched
	SideEffect
	OnlyHash
	NextPulseOnly
	Unblocked // ?
	Recap // can follow up any record (Deactivate?)
)

type RecordPolicy struct {
	FieldPolicy
	CanFollow RecordTypeSet
	RedirectTo RecordTypeSet
}

func (v RecordPolicy) IsValid() bool {
	return v.FieldPolicy != 0 || !v.CanFollow.IsZero()
}

func (v RecordPolicy) CheckRecordRef(lineBase reference.LocalHolder, localPN pulse.Number, ref reference.LocalHolder) error {
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

	if pn := refLocal.GetPulseNumber(); localPN != pn {
		return throw.E("wrong pulse number", struct { Expected, Actual pulse.Number }{ localPN, pn })
	}

	return nil
}

func (v RecordPolicy) CheckRootRef(ref reference.Holder, policyFn RecordPolicyProviderFunc, resolverFn PolicyResolverFunc) error {
	switch {
	case v.FieldPolicy&(LineStart|Branched) == LineStart:
		if !ref.IsEmpty() {
			return throw.E("must be empty")
		}
		switch found, err := resolverFn(ref); {
		case err != nil:
			return err
		case !found.IsZero():
			return throw.E("already exists")
		}
		return nil

	case ref.IsEmpty():
		return throw.E("must be not empty")
	}

	switch found, err := resolverFn(ref); {
	case err != nil:
		return err
	case found.IsZero():
		return throw.E("missing record")
	case found.RecordType == RecordNotAvailable:
	case policyFn == nil:
	default:
		msg := ""
		switch policy := policyFn(found.RecordType); {
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

func (v RecordPolicy) CheckPrevRef(pn pulse.Number, ref reference.Holder, resolverFn PolicyResolverFunc) error {
	switch {
	case v.FieldPolicy&LineStart != 0:
		if !ref.IsEmpty() {
			return throw.E("must be empty")
		}
		return nil
	case ref.IsEmpty():
		return throw.E("must be not empty")
	case v.FieldPolicy&NextPulseOnly == 0:
	case ref.GetLocal().GetPulseNumber() >= pn:
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

func (v RecordPolicy) CheckRejoinRef(pn pulse.Number, ref reference.Holder, resolverFn PolicyResolverFunc) error {
	switch {
	case v.FieldPolicy&SideEffect == 0:
		if !ref.IsEmpty() {
			return throw.E("must be empty")
		}
		return nil
	case ref.IsEmpty():
		return throw.E("must be not empty")
	case ref.GetLocal().GetPulseNumber() != pn:
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
	if ref.IsEmpty() {

	}
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

func (v RecordPolicy) IsFilamentStartNotBranched() bool {
	return v.FieldPolicy&(FilamentStart|Branched) == FilamentStart
}

func (v RecordPolicy) CanBeRejoined() bool {
	return v.FieldPolicy&(SideEffect|FilamentEnd) == FilamentEnd
}
