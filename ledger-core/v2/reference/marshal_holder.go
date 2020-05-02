// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func ProtoSize(h Holder) int {
	base := h.GetBase()
	if base.IsEmpty() {
		return ProtoSizeLocal(h.GetLocal())
	}
	return LocalBinarySize + ProtoSizeLocal(h.GetLocal())
}

func MarshalTo(h Holder, b []byte) (int, error) {
	base := h.GetBase()
	if base.IsEmpty() {
		return MarshalLocalTo(h.GetLocal(), b)
	}
	return _marshal(h.GetLocal(), base, b)
}

func Marshal(h Holder) ([]byte, error) {
	base := h.GetBase()
	if base.IsEmpty() {
		v := h.GetLocal()
		return MarshalLocal(v)
	}

	b := make([]byte, LocalBinarySize+ProtoSizeLocal(base))
	n, err := _marshal(h.GetLocal(), base, b)
	if err != nil {
		return nil, err
	}
	return b[:n], err
}

func Encode(h Holder) (string, error) {
	return DefaultEncoder().Encode(h)
}

func Decode(s string) (Holder, error) {
	return DefaultDecoder().DecodeHolder(s)
}

func MarshalJSON(h Holder) ([]byte, error) {
	if h == nil {
		return json.Marshal(nil)
	}
	s, err := Encode(h)
	if err != nil {
		return nil, err
	}
	return json.Marshal(s)
}

func UnmarshalJSON(b []byte) (v Global, err error) {
	err = _unmarshalJSON(&v, b)
	return
}

func _unmarshalJSON(v *Global, data []byte) error {
	var repr interface{}

	err := json.Unmarshal(data, &repr)
	if err != nil {
		return throw.W(err, "failed to unmarshal reference.Global")
	}

	switch realRepr := repr.(type) {
	case string:
		decoded, err := Decode(realRepr)
		if err != nil {
			return throw.W(err, "failed to unmarshal reference.Global")
		}

		v.addressLocal = decoded.GetLocal()
		v.addressBase = decoded.GetBase()
	case nil:
	default:
		return throw.W(err, fmt.Sprintf("unexpected type %T when unmarshal reference.Global", repr))
	}

	return nil
}

func _marshal(local, base Local, b []byte) (int, error) {
	n := WriteWholeLocalTo(local, b)
	if n != LocalBinarySize {
		return 0, throw.WithStackTop(io.ErrShortBuffer)
	}
	n2, err := MarshalLocalTo(base, b[n:])
	return n + n2, err
}

func Unmarshal(b []byte) (local, base Local, err error) {
	switch n := len(b); {
	case n == 0:
		return
	case n <= LocalBinaryPulseAndScopeSize:
		err = throw.WithStackTop(io.ErrShortBuffer)
		return
	case n <= LocalBinarySize:
		local, err = UnmarshalLocal(b)
		return
	case n > GlobalBinarySize:
		err = errors.New("too much bytes to unmarshal reference.Global")
		return
	default:
		local = ReadWholeLocalFrom(b[:LocalBinarySize])
		base, err = UnmarshalLocal(b[LocalBinarySize:])
		return
	}
}
