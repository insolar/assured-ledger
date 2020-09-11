// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

import (
	"io"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const (
	LocalBinaryHashSize          = 28
	LocalBinaryPulseAndScopeSize = 4
	LocalBinarySize              = LocalBinaryPulseAndScopeSize + LocalBinaryHashSize
	GlobalBinarySize             = 2 * LocalBinarySize
)

/* For LIMITED USE ONLY - can only be used by observer/analytical code */
func NewRecordRef(recID Local) Global {
	if recID.SubScope() != 0 {
		panic("illegal value")
	}
	return Global{addressLocal: recID}
}

func NewSelf(localID Local) Global {
	if localID.SubScope() == baseScopeReserved {
		panic("illegal value")
	}
	return Global{addressLocal: localID, addressBase: localID}
}

func New(domainID, localID Local) Global {
	return Global{addressLocal: localID, addressBase: domainID}
}

func NewRecordOf(owner Global, localID Local) Global {
	var (
		base      = owner.GetBase()
		basePulse = base.Pulse()
	)
	if base.IsEmpty() {
		panic(throw.IllegalValue())
	}
	if !basePulse.IsSpecial() && base.Pulse() > localID.Pulse() {
		panic(throw.IllegalValue())
	}
	return Global{addressLocal: localID, addressBase: base}
}

type Global struct {
	addressLocal Local
	addressBase  Local
}

func (v Global) IsZero() bool {
	return v.addressLocal.IsZero() && v.addressBase.IsZero()
}

func (v Global) IsEmpty() bool {
	return v.addressLocal.IsEmpty() && v.addressBase.IsEmpty()
}

func (v Global) GetScope() Scope {
	return v.addressBase.SubScope().AsBaseOf(v.addressLocal.SubScope())
}

func (v Global) WriteTo(w io.Writer) (int64, error) {
	n, err := v.addressLocal.WriteTo(w)
	if err != nil {
		return n, err
	}
	n2, err := v.addressBase.WriteTo(w)
	return n + n2, err
}

func (v Global) AsByteString() longbits.ByteString {
	return longbits.CopyBytes(v.AsBytes())
}

// deprecated // use reference.AsBytes()
func (v Global) AsBytes() []byte {
	return AsBytes(v)
}

func (v Global) AsRecordID() Local {
	return AsRecordID(v)
}

func (v Global) IsRecordScope() bool {
	return IsRecordScope(v)
}

func (v Global) IsObjectReference() bool {
	return IsObjectReference(v)
}

func (v Global) IsSelfScope() bool {
	return IsSelfScope(v)
}

func (v Global) IsLifelineScope() bool {
	return IsLifelineScope(v)
}

func (v Global) IsLocalDomainScope() bool {
	return IsLocalDomainScope(v)
}

func (v Global) IsGlobalScope() bool {
	return IsGlobalScope(v)
}

func (v Global) GetBase() Local {
	return v.addressBase
}

func (v Global) GetLocal() Local {
	return v.addressLocal
}

func (v Global) String() string {
	s, err := DefaultEncoder().Encode(v)
	if err != nil {
		return NilRef
	}
	return s
}

/* ONLY for parser */
func (v *Global) convertToSelf() {
	if !v.tryConvertToSelf() {
		panic("illegal state")
	}
}

/* ONLY for parser */
func (v *Global) tryConvertToSelf() bool {
	if !v.canConvertToSelf() {
		return false
	}
	v.addressBase = v.addressLocal
	return true
}

/* ONLY for parser */
func (v *Global) canConvertToSelf() bool {
	return v.addressBase.IsEmpty() && v.addressLocal.canConvertToSelf()
}

/* ONLY for parser */
func (v *Global) tryConvertToCompact() Holder {
	switch {
	case v.addressBase.IsEmpty():
		return NewPtrRecord(v.addressLocal)
	case v.addressLocal.IsEmpty():
		return NewPtrHolder(v.addressBase, v.addressLocal)
	case v.addressBase == v.addressLocal:
		return NewPtrSelf(v.addressLocal)
	default:
		return v
	}
}

/* ONLY for parser */
func (v *Global) tryApplyBase(base *Global) bool {
	if !v.addressBase.IsEmpty() {
		return false
	}

	if !base.IsSelfScope() {
		switch base.GetScope() {
		case LocalDomainMember, GlobalDomainMember:
			break
		default:
			return false
		}
	}
	v.addressBase = base.addressLocal
	return true
}

func (v Global) Compare(other Holder) int {
	return Compare(v, other)
}

func (v Global) Equal(other Holder) bool {
	return Equal(v, other)
}

// deprecated: use reference.Marshal, seems to be used by CBOR
func (v *Global) MarshalBinary() ([]byte, error) {
	return Marshal(v)
}

// deprecated: use reference.Unmarshal, seems to be used by CBOR
func (v *Global) UnmarshalBinary(data []byte) (err error) {
	if len(data) == 0 {
		*v = Global{}
		return nil
	}
	*v, err = UnmarshalGlobal(data)
	return
}

func GlobalFromString(input string) (Global, error) {
	global, err := DefaultDecoder().Decode(input)
	switch {
	case err != nil:
		return Global{}, err
	case global.addressBase.IsZero():
		// TODO enable when relevant logic in tests will be fixed
		// return Global{}, errors.New("base is zero")
	}

	return global, nil
}

func GlobalObjectFromString(input string) (Global, error) {
	global, err := GlobalFromString(input)
	if err != nil {
		return Global{}, err
	}
	if !global.IsObjectReference() {
		return Global{}, errors.New("provided reference is not object")
	}
	if !global.IsSelfScope() {
		return Global{}, errors.New("provided reference is not self-scoped")
	}
	return global, nil
}

func GlobalFromBytes(data []byte) (v Global) {
	var err error
	v, err = UnmarshalGlobal(data)
	if err != nil {
		panic(throw.IllegalValue())
	}
	return v
}
