// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package unsafekit

import (
	"reflect"
	"runtime"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/reference"
)

func TestRefMappable(t *testing.T) {
	var v reference.Local
	require.Equal(t, MemoryModelDepended, MemoryModelDependencyOf(reflect.TypeOf(v)))
}

func TestLocalRefSize(t *testing.T) {
	var v reference.Local
	if unsafe.Sizeof(v) != reference.LocalBinarySize {
		panic("unexpected alignment of reference.Local")
	}
}

func TestGlobalRefSize(t *testing.T) {
	var v reference.Global
	if unsafe.Sizeof(v) != reference.GlobalBinarySize {
		panic("unexpected alignment of reference.Global")
	}
}

func TestWrapLocalRef(t *testing.T) {
	if BigEndian {
		t.SkipNow()
	}

	binary := [reference.LocalBinarySize]byte{}
	binary[0] = 1
	binary[1] = 2
	binary[2] = 3
	binary[3] = 4
	ref := UnwrapAsLocalRef(WrapBytes(binary[:]))
	require.Equal(t, 0x04030201, int(ref.GetPulseNumber()))
	binary[0] = 0xFF
	require.Equal(t, 0x040302FF, int(ref.GetPulseNumber()))

	runtime.KeepAlive(binary)
}

func TestWrapAs(t *testing.T) {
	if BigEndian {
		t.SkipNow()
	}

	binary := [reference.LocalBinarySize]byte{}
	binary[0] = 1
	binary[1] = 2
	binary[2] = 3
	binary[3] = 4
	var v reference.Local

	mt := MustMMapType(reflect.TypeOf(v), false)
	require.NotNil(t, mt)

	ref := UnwrapAs(WrapBytes(binary[:]), mt).(*reference.Local)
	require.Equal(t, 0x04030201, int(ref.GetPulseNumber()))
	binary[0] = 0xFF
	require.Equal(t, 0x040302FF, int(ref.GetPulseNumber()))

	runtime.KeepAlive(binary)
}

func TestWrapGlobalRef(t *testing.T) {
	if BigEndian {
		t.SkipNow()
	}

	binary := [reference.GlobalBinarySize]byte{}
	binary[0] = 1
	binary[1] = 2
	binary[2] = 3
	binary[3] = 4
	ref := UnwrapAsGlobalRef(WrapBytes(binary[:]))
	require.Equal(t, 0x04030201, int(ref.GetLocal().GetPulseNumber()))
	require.Equal(t, 0, int(ref.GetBase().GetPulseNumber()))
	binary[0] = 0xFF
	require.Equal(t, 0x040302FF, int(ref.GetLocal().GetPulseNumber()))
	require.Equal(t, 0, int(ref.GetBase().GetPulseNumber()))

	runtime.KeepAlive(binary)
}
