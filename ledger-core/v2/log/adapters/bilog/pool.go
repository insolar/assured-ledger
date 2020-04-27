// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package bilog

import "sync"

const poolEntrySize = minEventBuffer

type poolBuffer *[poolEntrySize]byte

var bufferPool = &sync.Pool{
	New: func() interface{} {
		return poolBuffer(&[poolEntrySize]byte{})
	},
}

func allocateBuffer(capacity int) poolBuffer {
	switch {
	case capacity > poolEntrySize:
		return nil
	case capacity <= 0:
		panic("illegal value")
	}
	return bufferPool.Get().(poolBuffer)
}

func reuseBuffer(buf poolBuffer) {
	if buf == nil {
		return
	}
	bufferPool.Put(buf)
}
