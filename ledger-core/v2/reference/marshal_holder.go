// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func ProtoSize(h Holder) int {
	base := h.GetBase()
	if base.IsEmpty() {
		return h.GetLocal().ProtoSize()
	}
	return LocalBinarySize + base.ProtoSize()
}

func MarshalTo(h Holder, b []byte) (int, error) {
	base := h.GetBase()
	if base.IsEmpty() {
		return h.GetLocal().MarshalTo(b)
	}
	return _marshal(h.GetLocal(), base, b)
}

func Marshal(h Holder) ([]byte, error) {
	base := h.GetBase()
	if base.IsEmpty() {
		return h.GetLocal().Marshal()
	}

	b := make([]byte, LocalBinarySize+base.ProtoSize())
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

func UnmarshalJSON(b []byte) (Holder, error) {
	v := Global{}
	err := v.unmarshalJSON(b)
	return v, err
}

func _marshal(local, base *Local, b []byte) (int, error) {
	n, err := local.wholeMarshalTo(b)
	switch {
	case err != nil:
		return 0, err
	case n != LocalBinarySize:
		return 0, throw.WithStackTop(io.ErrUnexpectedEOF)
	}

	n2 := 0
	n2, err = base.MarshalTo(b[n:])
	return n + n2, err
}

func Unmarshal(b []byte) (local, base Local, err error) {
	switch n := len(b); {
	case n == 0:
		return
	case n <= LocalBinaryPulseAndScopeSize:
		err = throw.WithStackTop(io.ErrUnexpectedEOF)
		return
	case n <= LocalBinarySize:
		err = local.unmarshal(b)
		return
	case n > GlobalBinarySize:
		err = errors.New("too much bytes to unmarshal reference.Global")
		return
	default:
		err = local.wholeUnmarshalLocal(b[:LocalBinarySize])
		if err == nil {
			err = base.unmarshal(b[LocalBinarySize:])
		}
		return
	}
}
