// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package bilog

import "sync"

const poolEntrySize = minEventBuffer

var bufferPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, poolEntrySize)
	},
}

func allocateBuffer(capacity int) []byte {
	switch {
	case capacity > poolEntrySize:
		return make([]byte, 0, capacity)
	case capacity <= 0:
		panic("illegal value")
	}
	return bufferPool.Get().([]byte)[:0]
}

func reuseBuffer(buf []byte) {
	if cap(buf) != poolEntrySize {
		return
	}
	// TODO: probably it is better to switch to buffer or *[]byte
	// see details in https://staticcheck.io/docs/checks#SA6002
	bufferPool.Put(buf[:0]) // nolint:staticcheck
}
