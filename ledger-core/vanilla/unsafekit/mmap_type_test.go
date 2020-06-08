// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package unsafekit

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func mModelOf(v interface{}) MemoryMapModel {
	return MemoryModelDependencyOf(reflect.TypeOf(v))
}

func TestMemoryModelDependencyOf(t *testing.T) {
	require.Equal(t, MemoryModelIndependent, mModelOf(byte(0)))
	require.Equal(t, MemoryModelIndependent, mModelOf(struct {
	}{}))

	require.Equal(t, MemoryModelIndependent, mModelOf(struct {
		b byte
		f bool
	}{}))

	require.Equal(t, MemoryModelDepended, mModelOf(int(0)))
	require.Equal(t, MemoryModelDepended, mModelOf(struct {
		b byte
		f int
	}{}))

	require.Equal(t, MemoryMapIncompatible, mModelOf(&struct{}{}))
	require.Equal(t, MemoryMapIncompatible, mModelOf(struct {
		b byte
		f int
		p *struct{}
	}{}))
	require.Equal(t, MemoryMapIncompatible, mModelOf(""))
}
