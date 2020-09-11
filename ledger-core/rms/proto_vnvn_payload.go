// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"encoding/base64"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

const (
	MessageHashSize = 28
)

type MessageHash [MessageHashSize]byte

func (h *MessageHash) MarshalTo(data []byte) (int, error) {
	if len(data) < len(h) {
		return 0, throw.New("Not enough bytes to marshal PulseNumber")
	}
	copy(data, h[:])
	return len(h), nil
}

func (h *MessageHash) Unmarshal(data []byte) error {
	if len(data) < MessageHashSize {
		return throw.New("not enough bytes")
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
func UnmarshalFromMeta(meta []byte) (Marshaler, error) {
	m := Meta{}
	err := m.Unmarshal(meta)
	if err != nil {
		return nil, err
	}
	pl := m.Payload.Get()
	return pl.(Marshaler), nil
}
