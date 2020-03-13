// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

import (
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
	if n, err := _marshal(h.GetLocal(), base, b); err != nil {
		return nil, err
	} else {
		return b[:n], err
	}
}

func _marshal(local, base *Local, b []byte) (int, error) {
	n, err := local.FullMarshalTo(b)
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

func Unmarshal(local, base *Local, b []byte) error {
	switch n := len(b); {
	case n == 0:
		*local = Local{}
		*base = Local{}
		return nil
	case n <= LocalBinaryPulseAndScopeSize:
		return throw.WithStackTop(io.ErrUnexpectedEOF)
	case n <= LocalBinarySize:
		*base = Local{}
		return local.Unmarshal(b)
	case n > GlobalBinarySize:
		return errors.New("too much bytes to unmarshal reference.Global")
	default:
		if err := local.FullUnmarshal(b[:LocalBinarySize]); err != nil {
			return err
		}
		return base.Unmarshal(b[LocalBinarySize:])
	}
}
