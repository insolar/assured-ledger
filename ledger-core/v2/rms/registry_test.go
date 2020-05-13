// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	prev := GetRegistry()
	defer SetRegistry(prev)

	temp := &TypeRegistry{}
	SetRegistry(temp)
	require.Panics(t, func() { SetRegistry(nil) })
	require.Equal(t, temp, GetRegistry())

	require.Panics(t, func() { temp.Put(1, nil) })
	require.Panics(t, func() { temp.Put(1, reflect.TypeOf(0)) })

	temp.Put(999999990, reflect.TypeOf((*MessageExample)(nil)))
	require.Panics(t, func() { temp.Put(999999990, reflect.TypeOf((*MessageExample)(nil))) })
	require.Panics(t, func() { temp.Put(0, reflect.TypeOf((*MessageExample)(nil))) })
	temp.Put(1, reflect.TypeOf((*MessageExample)(nil)))

	temp.PutSpecial(999999990, "Head", reflect.TypeOf((*MessageExample_Head)(nil)))
	temp.PutSpecial(999999990, "HeadX", reflect.TypeOf((*MessageExample_Head)(nil)))
	require.Panics(t, func() { temp.PutSpecial(999999990, "Head", reflect.TypeOf((*MessageExample_Head)(nil))) })
	require.Panics(t, func() { temp.PutSpecial(999999990, "", reflect.TypeOf((*MessageExample_Head)(nil))) })
	require.Panics(t, func() { temp.PutSpecial(0, "", reflect.TypeOf((*MessageExample_Head)(nil))) })

	require.Nil(t, temp.Get(0))
	require.Nil(t, temp.Get(99))
	require.Nil(t, temp.GetSpecial(0, "Head"))
	require.Nil(t, temp.GetSpecial(99, "Head"))

	require.Equal(t, reflect.TypeOf((*MessageExample)(nil)).Elem(), temp.Get(1))
	require.Equal(t, reflect.TypeOf((*MessageExample)(nil)).Elem(), temp.Get(999999990))
	require.Equal(t, reflect.TypeOf((*MessageExample)(nil)).Elem(), temp.GetSpecial(1, ""))
	require.Equal(t, reflect.TypeOf((*MessageExample)(nil)).Elem(), temp.GetSpecial(999999990, ""))

	require.Nil(t, temp.GetSpecial(1, "Head"))
	require.Equal(t, reflect.TypeOf((*MessageExample_Head)(nil)).Elem(), temp.GetSpecial(999999990, "Head"))

	id, m, err := Unmarshal([]byte{0x80, 0x1, 0x1})
	require.NoError(t, err)
	require.Equal(t, uint64(1), id)
	require.Equal(t, &MessageExample{}, m)

	id, m, err = Unmarshal([]byte{0x80, 0x1, 0})
	require.Error(t, err)

	id, m, err = Unmarshal([]byte{0x08, 0x1, 1})
	require.Error(t, err)

	id, m, err = UnmarshalSpecial([]byte{0x80, 0x1, 1}, "Head")
	require.Error(t, err)

	temp.PutSpecial(1, "Head", reflect.TypeOf((*MessageExample_Head)(nil)))

	id, m, err = UnmarshalSpecial([]byte{0x80, 0x1, 1}, "Head")
	require.NoError(t, err)
	require.Equal(t, uint64(1), id)
	require.Equal(t, &MessageExample_Head{}, m)
}
