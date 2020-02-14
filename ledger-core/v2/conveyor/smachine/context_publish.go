// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package smachine

import (
	"reflect"

	"github.com/insolar/assured-ledger/ledger-core/v2/pulse"
	"github.com/insolar/assured-ledger/ledger-core/v2/vanilla/longbits"
)

type globalAliasKey struct {
	key interface{}
}

func isValidPublishValue(data interface{}) bool {
	switch data.(type) {
	case nil, dependencyKey, slotIdKey, *slotAliasesValue, *uniqueAliasKey, globalAliasKey:
		return false
	}
	return true
}

func isValidPublishKey(key interface{}) bool {
	switch key.(type) {
	case nil, dependencyKey, slotIdKey, *slotAliasesValue, *uniqueAliasKey, globalAliasKey:
		return false
	case bool, int8, int16, int32, int64, int, uint8, uint16, uint32, uint64, uint, uintptr:
		return true
	case float32, float64, complex64, complex128, string:
		return true
	case longbits.ByteString, longbits.Bits64, longbits.Bits128, longbits.Bits224, longbits.Bits256, longbits.Bits512:
		return true
	case pulse.Number, SlotID:
		return true
	default:
		// have to go for reflection
		switch tt := reflect.TypeOf(key).Kind(); {
		case tt <= reflect.Array: // literals
			return tt > reflect.Invalid
		case tt >= reflect.String: // String, Struct, UnsafePointer
			return tt <= reflect.UnsafePointer
		case tt == reflect.Ptr:
			return true
		default: // Chan, Func, Interface, Map, Slice
			return false
		}
	}
}

func ensurePublishValue(data interface{}) {
	if !isValidPublishValue(data) {
		panic("illegal value")
	}
}

func ensureShareValue(data interface{}) {
	if !isValidPublishValue(data) {
		panic("illegal value")
	}
	switch data.(type) {
	case SharedDataLink, *SharedDataLink:
		panic("illegal value - SharedDataLink can't be shared")
	}
}

func ensurePublishKey(key interface{}) {
	if !isValidPublishKey(key) {
		panic("illegal value")
	}
}
