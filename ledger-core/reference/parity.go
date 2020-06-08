// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package reference

import (
	"bytes"
	"errors"
	"hash/fnv"
)

func GetParity(ref Holder) []byte {
	hasher := fnv.New32a()
	_, _ = ref.GetLocal().WriteTo(hasher)
	_, _ = ref.GetBase().WriteTo(hasher)
	return hasher.Sum(nil)[0:2] // 3 bytes
}

func CheckParity(ref Holder, b []byte) error {
	if bytes.Equal(b, GetParity(ref)) {
		return nil
	}
	return errors.New("parity mismatch")
}
