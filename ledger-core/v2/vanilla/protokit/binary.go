// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package protokit

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/throw"
)

func BinaryProtoSize(n int) int {
	if n > 0 {
		return n + 1
	}
	return n
}

func BinaryMarshalTo(b []byte, marshalTo func([]byte) (int, error)) (int, error) {
	switch {
	case len(b) > 0:
		b[0] = BinaryMarker
		return marshalTo(b[1:])
	default:
		// make sure that buf size was checked
		return marshalTo(nil)
	}
}

func BinaryMarshalToSizedBuffer(b []byte, marshalToSizedBuffer func([]byte) (int, error)) (int, error) {
	switch n, err := marshalToSizedBuffer(b); {
	case err != nil || n == 0:
		return n, err
	case n == 0:
		return 0, nil
	case n == len(b):
		return 0, io.ErrShortBuffer
	default:
		n++
		b[len(b)-n] = BinaryMarker
		return n, nil
	}
}

func BinaryUnmarshal(b []byte, unmarshal func([]byte) error) error {
	if len(b) == 0 {
		return unmarshal(nil)
	}
	if b[0] != BinaryMarker {
		return throw.FailHere("expected binary marker")
	}
	return unmarshal(b[1:])
}
