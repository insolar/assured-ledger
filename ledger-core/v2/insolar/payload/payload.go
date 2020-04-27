// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"encoding/base64"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

type Type uint32

//go:generate stringer -type=Type
//go:generate protoc -I=. -I=$GOPATH/src --gogoslick_out=./ payload.proto

const (
	TypeUnknown Type = iota

	TypeMeta
	TypeError
	TypeID
	TypeIDs

	// New virtual message types
	TypeVCallRequest
	TypeVCallResult
	TypeVStateReport
	TypeVStateRequest
	TypeVPendingDelegationRequest
	TypeVDelegatedRequestFinished

	// should be the last (required by TypesMap)
	_latestType
)

// TypesMap contains Type name (gen by stringer) to type mapping.
var TypesMap = func() map[string]Type {
	m := map[string]Type{}
	for i := TypeUnknown; i < _latestType; i++ {
		m[i.String()] = i
	}
	return m
}()

// Payload represents any kind of data that can be encoded in consistent manner.
type Payload interface {
	Marshal() ([]byte, error)
}

const (
	MessageHashSize = 28
	MorphFieldNum   = 16
	MorpyFieldType  = 0 // Varint
)

type MessageHash [MessageHashSize]byte

func (h *MessageHash) MarshalTo(data []byte) (int, error) {
	if len(data) < len(h) {
		return 0, errors.New("Not enough bytes to marshal PulseNumber")
	}
	copy(data, h[:])
	return len(h), nil
}

func (h *MessageHash) Unmarshal(data []byte) error {
	if len(data) < MessageHashSize {
		return errors.New("not enough bytes")
	}
	copy(h[:], data)
	return nil
}

func (h MessageHash) Equal(other MessageHash) bool {
	return h == other
}

func (h MessageHash) Size() int {
	return len(h)
}

func (h *MessageHash) String() string {
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func (h *MessageHash) IsZero() bool {
	for _, b := range h {
		if b != 0 {
			return false
		}
	}
	return true
}

// UnmarshalType decodes payload type from given binary.
func UnmarshalType(data []byte) (Type, error) {
	buf := proto.NewBuffer(data)
	fieldNumType, err := buf.DecodeVarint()
	if err != nil {
		return TypeUnknown, errors.Wrap(err, "failed to decode polymorph")
	}
	// First 3 bits is a field type (see protobuf wire protocol docs), key is always varint
	if fieldNumType != MorphFieldNum<<3|MorpyFieldType {
		return TypeUnknown, errors.Errorf("wrong polymorph field number %d", fieldNumType)
	}
	morph, err := buf.DecodeVarint()
	if err != nil {
		return TypeUnknown, errors.Wrap(err, "failed to decode polymorph")
	}
	return Type(morph), nil
}

// MarshalType encodes payload type into binary.
func MarshalType(t Type) ([]byte, error) {
	buf := proto.NewBuffer(nil)
	err := buf.EncodeVarint(MorphFieldNum<<3 | MorpyFieldType)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode polymorph")
	}
	err = buf.EncodeVarint(uint64(t))
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode polymorph")
	}
	return buf.Bytes(), nil
}

func Marshal(payload Payload) ([]byte, error) {
	switch pl := payload.(type) {
	case *Meta:
		pl.Polymorph = uint32(TypeMeta)
		return pl.Marshal()
	case *Error:
		pl.Polymorph = uint32(TypeError)
		return pl.Marshal()
	case *ID:
		pl.Polymorph = uint32(TypeID)
		return pl.Marshal()
	case *IDs:
		pl.Polymorph = uint32(TypeIDs)
		return pl.Marshal()
	case *VCallRequest:
		pl.Polymorph = uint32(TypeVCallRequest)
		return pl.Marshal()
	case *VCallResult:
		pl.Polymorph = uint32(TypeVCallResult)
		return pl.Marshal()
	case *VStateRequest:
		pl.Polymorph = uint32(TypeVStateRequest)
		return pl.Marshal()
	case *VStateReport:
		pl.Polymorph = uint32(TypeVStateReport)
		return pl.Marshal()
	case *VPendingDelegationRequest:
		pl.Polymorph = uint32(TypeVPendingDelegationRequest)
		return pl.Marshal()
	case *VDelegatedRequestFinished:
		pl.Polymorph = uint32(TypeVDelegatedRequestFinished)
		return pl.Marshal()
	}

	return nil, errors.New("unknown payload type")
}

func MustMarshal(p Payload) []byte {
	buf, err := Marshal(p)
	if err != nil {
		panic(err)
	}
	return buf
}

func Unmarshal(data []byte) (Payload, error) {
	tp, err := UnmarshalType(data)
	if err != nil {
		return nil, err
	}
	switch tp {
	case TypeMeta:
		pl := Meta{}
		err := pl.Unmarshal(data)
		return &pl, err
	case TypeError:
		pl := Error{}
		err := pl.Unmarshal(data)
		return &pl, err
	case TypeID:
		pl := ID{}
		err := pl.Unmarshal(data)
		return &pl, err
	case TypeIDs:
		pl := IDs{}
		err := pl.Unmarshal(data)
		return &pl, err
	case TypeVCallRequest:
		pl := VCallRequest{}
		err := pl.Unmarshal(data)
		return &pl, err
	case TypeVCallResult:
		pl := VCallResult{}
		err := pl.Unmarshal(data)
		return &pl, err
	case TypeVStateRequest:
		pl := VStateRequest{}
		err := pl.Unmarshal(data)
		return &pl, err
	case TypeVStateReport:
		pl := VStateReport{}
		err := pl.Unmarshal(data)
		return &pl, err
	case TypeVPendingDelegationRequest:
		pl := VPendingDelegationRequest{}
		err := pl.Unmarshal(data)
		return &pl, err
	case TypeVDelegatedRequestFinished:
		pl := VDelegatedRequestFinished{}
		err := pl.Unmarshal(data)
		return &pl, err
	}

	return nil, errors.New("unknown payload type")
}

// UnmarshalFromMeta reads only payload skipping meta decoding. Use this instead of regular Unmarshal if you don't need
// Meta data.
func UnmarshalFromMeta(meta []byte) (Payload, error) {
	m := Meta{}
	// Can be optimized by using proto.NewBuffer.
	err := m.Unmarshal(meta)
	if err != nil {
		return nil, err
	}
	pl, err := Unmarshal(m.Payload)
	if err != nil {
		return nil, err
	}

	return pl, nil
}
