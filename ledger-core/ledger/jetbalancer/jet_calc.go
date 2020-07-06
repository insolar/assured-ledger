// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package jetbalancer

import (
	"encoding/binary"
	"io"

	"github.com/insolar/x-crypto/sha256"
)

type MetricCalc struct {}

func (v MetricCalc) CalcMetric(prefix uint64, entropy io.WriterTo) uint64 {
	encoding := binary.LittleEndian

	hash := sha256.New()
	var b [8]byte
	encoding.PutUint32(b[:4], uint32(prefix))
	hash.Write(b[:4])
	_, _ = entropy.WriteTo(hash)
	hash.Sum(b[:0])
	return encoding.Uint64(b[:])
}
