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
