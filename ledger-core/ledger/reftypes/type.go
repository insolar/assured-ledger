// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reftypes

import (
	"github.com/insolar/assured-ledger/ledger-core/pulse"
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type RefType uint32

const subTypeShift = 17
const subType RefType = 1<<subTypeShift

const (
	Zero           RefType = 0
	Invalid        RefType = 1
	PackagePrivate RefType = 2

	Jet = RefType(pulse.Jet)
	JetRecord = Jet + subType
	JetLeg = Jet + subType * 2
	JetDrop = Jet + subType * 3

	// TODO JetContract

	BuiltinContract = RefType(pulse.BuiltinContract)

	Node = RefType(pulse.Node)
	NodeContract = Node + subType

	RecordPayload = RefType(pulse.RecordPayload)

	APICall = RefType(pulse.ExternalCall)
	APISession128 = APICall + subType
	APISession384 = APICall + subType * 2

	APIEndpoint = RefType(pulse.EndpointAddress)
	// APITollFreeEndpoint = RefType(pulse.TollFreeEndpointAddress)

	// Pulse, Population etc

	Object RefType = pulse.MinTimePulse
)

type Usage uint8

func (v RefType) Usage() Usage {
	if typeDef := v.def(); typeDef != nil {
		return typeDef.Usage()
	}
	return 0
}

func (v RefType) def() RefTypeDef {
	return typeDefinition(v)
}

func (v RefType) SPN() pulse.Number {
	return pulse.Number(v.BaseType())
}

func (v RefType) BaseType() RefType {
	return v&0xFFFF
}

func (v RefType) SubType() uint32 {
	return uint32(v>>subTypeShift)
}

func (v RefType) IsDerivedType() bool {
	return v.SubType() != 0
}

func (v RefType) IsRelated(ref reference.Holder) bool {
	return !v.IsDerivedType() && v == TypeOf(ref).BaseType()
}

func (v RefType) Is(ref reference.Holder) bool {
	switch v {
	case Zero:
		return !reference.IsEmpty(ref)
	case Invalid:
		panic(throw.Unsupported())
	}

	if reference.IsEmpty(ref) {
		return false
	}

	_, err := v.inspect(v.def(), ref.GetBase(), ref.GetLocal())
	return err == nil
}

// func (v RefType) Of(ref reference.Holder) reference.Global {
// }
//
// func (v RefType) DeriveOf(ref reference.Holder, local reference.LocalHolder) reference.Global {
// }

func (v RefType) Inspect(ref reference.Holder) (base, local reference.Local, err error) {
	switch v {
	case Zero:
		if !reference.IsEmpty(ref) {
			return base, local, ErrNotEmptyRef
		}
	case Invalid:
		panic(throw.Unsupported())
	}

	if reference.IsEmpty(ref) {
		return base, local, ErrEmptyRef
	}

	base, local = ref.GetBase(), ref.GetLocal()
	checkLocal := false

	typeDef := v.def()
	switch checkLocal, err = v.inspect(typeDef, base, local); {
	case err != nil:
		err = newRefTypeErr(err, v, base, local)
	case checkLocal:
		err = typeDef.VerifyLocalRef(local)
	default:
		err = typeDef.VerifyGlobalRef(base, local)
	}
	return base, local, err
}

func (v RefType) inspect(typeDef RefTypeDef, base, local reference.Local) (bool, error) {
	bpn, lpn := base.Pulse(), local.Pulse()

	switch {
	case v == PackagePrivate:
		if bpn.IsPackagePrivate() || lpn.IsPackagePrivate() {
			return false, nil
		}
		return false, ErrIllegalRefValue
	case lpn == pulse.Unknown:
		return false, ErrInvalidRef
	case typeDef == nil:
		return false, ErrInvalidRef
	}

	switch usage := typeDef.Usage(); {
	case bpn != v.SPN():
		switch {
		case lpn != v.SPN():
		case usage & UseAsLocal == 0:
		case bpn.IsTimePulse():
			return true, nil
		}
	case bpn != lpn:
		if usage & UseAsBase != 0 {
			return false, nil
		}
	case base == local:
		if usage & UseAsSelf != 0 {
			return false, nil
		}
	case usage & (UseAsBase|UseAsLocal) == (UseAsBase|UseAsLocal):
		return false, nil
	}

	return false, ErrIllegalRefValue
}

func (v RefType) detectType(base, local reference.Local) RefType {
	typeDef := v.def()
	if typeDef == nil {
		return Zero
	}
	switch subType := typeDef.DetectSubType(base, local); subType {
	case Zero, v:
		return v
	case Invalid:
		return Zero
	default:
		if subType.BaseType() != v {
			panic(throw.IllegalState())
		}
		return subType
	}
}

func (v RefType) detectLocalType(usage Usage, local reference.Local) RefType {
	if typeDef := v.def(); typeDef != nil {
		if typeDef.Usage() & usage == usage {
			if t := typeDef.DetectSubType(reference.Local{}, local); t >= pulse.MaxTimePulse {
				return t
			}
		}
	}
	return Zero
}

func TypeOf(ref reference.Holder) RefType {
	if ref == nil {
		return Zero
	}

	base, local := ref.GetBase(), ref.GetLocal()
	bpn, lpn := base.Pulse(), local.Pulse()

	switch {
	case bpn == lpn:
		// possible self-ref or zero
		switch {
		case bpn.IsTimePulse():
			return Object
		case bpn.IsSpecial():
			if t := RefType(bpn).detectType(base, local); t != Zero {
				return t
			}
		case bpn == pulse.Unknown:
			return Zero
		case bpn.IsPackagePrivate():
			return PackagePrivate
		}
	case bpn.IsPackagePrivate() || lpn.IsPackagePrivate():
		return PackagePrivate
	case lpn == pulse.Unknown:
		return Invalid
	case bpn.IsTimePulse():
		if lpn.IsTimePulse() {
			return Object
		}
		if lpn.IsSpecial() {
			if t := RefType(bpn).detectLocalType(UseAsLocal, local); t != Zero {
				return t
			}
		}
	case bpn.IsSpecial():
		if t := RefType(bpn).detectType(base, local); t != Zero {
			return t
		}
	case bpn == pulse.Unknown:
		if t := RefType(bpn).detectLocalType(UseAsLocalValue, local); t != Zero {
			return t
		}
	}

	return Invalid
}
