// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package injector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDefaultInterfaceInjectionIDByPtr(t *testing.T) {
	assert.Panics(t, func() { _, _ = GetInterfaceTypeAndValue(nil) })
	type TestInterface interface{}
	var v TestInterface
	assert.Panics(t, func() { _, _ = GetInterfaceTypeAndValue(v) })
	assert.Panics(t, func() { _, _ = GetInterfaceTypeAndValue(&v) })
	v = struct{}{}
	_, vt := GetInterfaceTypeAndValue(&v)
	assert.Equal(t, GetDefaultInjectionIDByType(vt), "injector.TestInterface")
	assert.Panics(t, func() { _, _ = GetInterfaceTypeAndValue(v) })
	type TestStruct struct{}
	var s = TestStruct{}
	assert.Panics(t, func() { _, _ = GetInterfaceTypeAndValue(&s) })
}
