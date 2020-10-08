// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package protokit

import (
	"io"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

func BinaryProtoSize(n int) int {
	if n > 0 {
		return n + 1
	}
	return n
}

func BinaryMarshalTo(b []byte, allowEmpty bool, marshalTo func([]byte) (int, error)) (int, error) {
	if len(b) == 0 {
		return marshalTo(nil)
	}
	b[0] = BinaryMarker
	switch n, err := marshalTo(b[1:]); {
	case err != nil:
		return 0, err
	case !allowEmpty && n == 0:
		return 0, nil
	default:
		return n + 1, nil
	}
}

func BinaryMarshalToSizedBuffer(b []byte, allowEmpty bool, marshalToSizedBuffer func([]byte) (int, error)) (int, error) {
	switch n, err := marshalToSizedBuffer(b[1:]); {
	case err != nil:
		return 0, err
	case !allowEmpty && n == 0:
		return 0, nil
	default:
		n++
		i := len(b) - n
		if i < 0 {
			return 0, io.ErrShortBuffer
		}
		b[i] = BinaryMarker
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

const ExplicitEmptyBinaryProtoSize = 1
