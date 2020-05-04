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

func BinarySize(h Holder) int {
	if h == nil {
		return 0
	}
	base := h.GetBase()
	if base.IsEmpty() {
		return BinarySizeLocal(h.GetLocal())
	}
	return LocalBinarySize + BinarySizeLocal(h.GetLocal())
}

func MarshalTo(h Holder, b []byte) (int, error) {
	if h == nil {
		return 0, nil
	}
	base := h.GetBase()
	if base.IsEmpty() {
		return MarshalLocalTo(h.GetLocal(), b)
	}
	return _marshal(base, h.GetLocal(), b)
}

func MarshalToSizedBuffer(h Holder, b []byte) (int, error) {
	if h == nil {
		return 0, nil
	}
	base := h.GetBase()
	if base.IsEmpty() {
		return MarshalLocalToSizedBuffer(h.GetLocal(), b)
	}

	n, err := MarshalLocalToSizedBuffer(h.GetBase(), b)
	if err != nil {
		return n, err
	}
	if len(b) < LocalBinarySize+n {
		return n, throw.WithStackTop(io.ErrShortBuffer)
	}
	WriteWholeLocalTo(h.GetLocal(), b[:len(b)-n])
	return LocalBinarySize + n, nil
}

func Marshal(h Holder) ([]byte, error) {
	if h == nil {
		return nil, nil
	}
	base := h.GetBase()
	if base.IsEmpty() {
		v := h.GetLocal()
		return MarshalLocal(v)
	}

	b := make([]byte, LocalBinarySize+BinarySizeLocal(base))
	n, err := _marshal(base, h.GetLocal(), b)
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

func _marshal(base, local Local, b []byte) (int, error) {
	n := WriteWholeLocalTo(local, b)
	if n != LocalBinarySize {
		return 0, throw.WithStackTop(io.ErrShortBuffer)
	}
	n2, err := MarshalLocalTo(base, b[n:])
	return n + n2, err
}

func Unmarshal(b []byte) (base, local Local, err error) {
	var v Global
	v, err = UnmarshalGlobal(b)
	return v.addressBase, v.addressLocal, err
}

func UnmarshalGlobal(b []byte) (v Global, err error) {
	switch n := len(b); {
	case n == 0:
		return
	case n <= LocalBinaryPulseAndScopeSize:
		err = throw.WithStackTop(io.ErrShortBuffer)
		return
	case n <= LocalBinarySize:
		v.addressLocal, err = UnmarshalLocal(b)
		return
	case n > GlobalBinarySize:
		err = errors.New("too much bytes to unmarshal reference.Global")
		return
	default:
		v.addressLocal = ReadWholeLocalFrom(b[:LocalBinarySize])
		v.addressBase, err = UnmarshalLocal(b[LocalBinarySize:])
		return
	}
}
