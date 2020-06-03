// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package atomickit

import (
	"reflect"
	"sync/atomic"
	"unsafe"

	"github.com/insolar/assured-ledger/ledger-core/vanilla/longbits"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

type OnceByteString struct {
	len Int
	str unsafe.Pointer
}

func (p *OnceByteString) Load() longbits.ByteString {
	ln := p.len.Load()
	if ln <= 0 {
		return ""
	}
	str := atomic.LoadPointer(&p.str)
	if str == nil {
		panic(throw.IllegalState())
	}

	var result string
	resultHeader := (*reflect.StringHeader)(unsafe.Pointer(&result))
	resultHeader.Len = ln
	resultHeader.Data = uintptr(str)

	return longbits.WrapStr(result)
}

func (p *OnceByteString) StoreOnce(v longbits.ByteString) bool {
	switch header := (*reflect.StringHeader)(unsafe.Pointer(&v)); {
	case header.Len == 0:
		panic(throw.IllegalValue())
		//return p.len.Load() == 0
	case !p.len.CompareAndSwap(0, -1):
		return false
	default:
		atomic.StorePointer(&p.str, unsafe.Pointer(header.Data))
		p.len.Store(header.Len)
		return true
	}
}

func (p *OnceByteString) MustStore(v longbits.ByteString) {
	if !p.StoreOnce(v) {
		panic(throw.IllegalState())
	}
}

func (p *OnceByteString) CompareAndStore(new longbits.ByteString) bool {
	switch header := (*reflect.StringHeader)(unsafe.Pointer(&new)); {
	case header.Len == 0:
		return p.len.Load() == 0
	case !p.len.CompareAndSwap(0, -1):
		return p.Load() == new
	default:
		atomic.StorePointer(&p.str, unsafe.Pointer(header.Data))
		p.len.Store(header.Len)
		return true
	}
}

func (p *OnceByteString) String() string {
	return p.Load().String()
}
