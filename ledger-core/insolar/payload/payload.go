// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"encoding/base64"

	"github.com/gogo/protobuf/proto"

	"github.com/insolar/assured-ledger/ledger-core/rms"
	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// aliases for more convenient using in .proto file (no need to use full path)
// because alias in the same package
type MessageContext = rms.MessageContext

func RegisterMessageType(id uint64, special string, t proto.Message) {
	rms.RegisterMessageType(id, special, t)
}

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

// UnmarshalFromMeta reads only payload skipping meta decoding. Use this instead of regular Unmarshal if you don't need
// Meta data.
func UnmarshalFromMeta(meta []byte) (Payload, error) {
	m := Meta{}
	// Can be optimized by using proto.NewBuffer.
	err := m.Unmarshal(meta)
	if err != nil {
		return nil, err
	}
	_, pl, err := rms.Unmarshal(m.Payload)
	if err != nil {
		return nil, err
	}
	return pl.(Payload), nil
}
