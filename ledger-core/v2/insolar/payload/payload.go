// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"encoding/base64"

	"github.com/gogo/protobuf/proto"

	"github.com/insolar/assured-ledger/ledger-core/v2/rms"
	errors "github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

// aliases for more convenient using in .proto file (no need to use full path)
// because alias in the same package
type MessageContext = rms.MessageContext

func RegisterMessageType(id uint64, special string, t proto.Message) {
	rms.RegisterMessageType(id, special, t)
}

type Type uint32

//go:generate stringer -type=Type
// xgo:generate protoc -I=. -I=$GOPATH/src --gogoslick_out=./ payload.proto

const (
	TypeUnknown Type = iota

	TypeMeta

	// New virtual message types
	TypeVCallRequest
	TypeVCallResult
	TypeVStateReport
	TypeVStateUnavailable
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

// deprecated
func (h MessageHash) ProtoSize() int {
	return h.Size()
}

// UnmarshalType decodes payload type from given binary.
func UnmarshalType(data []byte) (Type, error) {
	buf := proto.NewBuffer(data)
	fieldNumType, err := buf.DecodeVarint()
	if err != nil {
		return TypeUnknown, errors.W(err, "failed to decode polymorph")
	}
	// First 3 bits is a field type (see protobuf wire protocol docs), key is always varint
	if fieldNumType != MorphFieldNum<<3|MorpyFieldType {
		return TypeUnknown, errors.Errorf("wrong polymorph field number %d", fieldNumType)
	}
	morph, err := buf.DecodeVarint()
	if err != nil {
		return TypeUnknown, errors.W(err, "failed to decode polymorph")
	}
	return Type(morph), nil
}

// MarshalType encodes payload type into binary.
func MarshalType(t Type) ([]byte, error) {
	buf := proto.NewBuffer(nil)
	err := buf.EncodeVarint(MorphFieldNum<<3 | MorpyFieldType)
	if err != nil {
		return nil, errors.W(err, "failed to encode polymorph")
	}
	err = buf.EncodeVarint(uint64(t))
	if err != nil {
		return nil, errors.W(err, "failed to encode polymorph")
	}
	return buf.Bytes(), nil
}

func Marshal(payload Payload) ([]byte, error) {
	return payload.Marshal()
}

func Unmarshal(data []byte) (Payload, error) {
	_, payload, err := rms.Unmarshal(data)

	return payload.(Payload), err
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
