// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
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

func NewGlobalSelf(localID Local) Global {
	if localID.SubScope() == baseScopeReserved {
		panic("illegal value")
	}
	return Global{addressLocal: localID, addressBase: localID}
}

func NewGlobal(domainID, localID Local) Global {
	return Global{addressLocal: localID, addressBase: domainID}
}

type Global struct {
	addressLocal Local
	addressBase  Local
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

func (v Global) Read(b []byte) (int, error) {
	return ReadTo(v, b)
}

func (v Global) AsByteString() longbits.ByteString {
	return longbits.CopyBytes(v.AsBytes())
}

func (v Global) AsBytes() []byte {
	val := make([]byte, GlobalBinarySize)
	_, _ = v.addressLocal.Read(val)
	_, _ = v.addressBase.Read(val[LocalBinarySize:])
	return val

}

func (v Global) IsEmpty() bool {
	return v.addressLocal.IsEmpty() && v.addressBase.IsEmpty()
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

func (v *Global) tryConvertToCompact() Holder {
	switch {
	case v.addressBase.IsEmpty():
		return NewRecord(v.addressLocal)
	case v.addressLocal.IsEmpty():
		return New(v.addressLocal, v.addressBase)
	case v.addressBase == v.addressLocal:
		return NewSelf(v.addressLocal)
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

func (v Global) GetBase() *Local {
	return &v.addressBase
}

func (v Global) GetLocal() *Local {
	return &v.addressLocal
}

func (v Global) String() string {
	s, err := DefaultEncoder().Encode(v)
	if err != nil {
		return NilRef
	}
	return s
}

func (v Global) Bytes() []byte {
	return v.AsBytes()
}

// deprecated
func (v Global) Equal(other Global) bool {
	return Equal(v, other)
}

func (v Global) Compare(other Global) int {
	return Compare(v, other)
}

// deprecated
func (v *Global) MarshalJSON() ([]byte, error) {
	return MarshalJSON(v)
}

// deprecated
func (v *Global) Marshal() ([]byte, error) {
	return Marshal(v)
}

// deprecated
func (v *Global) MarshalBinary() ([]byte, error) {
	return Marshal(v)
}

// deprecated
func (v *Global) MarshalTo(b []byte) (int, error) {
	return MarshalTo(v, b)
}

// deprecated
func (v *Global) UnmarshalJSON(data []byte) error {
	return v.unmarshalJSON(data)
}

func (v *Global) unmarshalJSON(data []byte) error {
	var repr interface{}

	err := json.Unmarshal(data, &repr)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal reference.Global")
	}

	switch realRepr := repr.(type) {
	case string:
		decoded, err := DefaultDecoder().Decode(realRepr)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal reference.Global")
		}

		v.addressLocal = *decoded.GetLocal()
		v.addressBase = *decoded.GetBase()
	case nil:
	default:
		return errors.Wrapf(err, "unexpected type %T when unmarshal reference.Global", repr)
	}

	return nil
}

// deprecated
func (v *Global) Unmarshal(data []byte) (err error) {
	v.addressLocal, v.addressBase, err = Unmarshal(data)
	return
}

// deprecated
func (v *Global) UnmarshalBinary(data []byte) error {
	return v.Unmarshal(data)
}

// deprecated
func (v Global) Size() int {
	return ProtoSize(v)
}

// deprecated
func (v *Global) ProtoSize() int {
	return ProtoSize(v)
}
