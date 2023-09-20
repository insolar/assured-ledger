package rms_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
)

func TestRegistry(t *testing.T) {
	prev := rmsreg.GetRegistry()
	defer rmsreg.SetRegistry(prev)

	temp := &rmsreg.TypeRegistry{}
	rmsreg.SetRegistry(temp)
	require.Panics(t, func() { rmsreg.SetRegistry(nil) })
	require.Equal(t, temp, rmsreg.GetRegistry())

	require.Panics(t, func() { temp.Put(1, nil) })
	require.Panics(t, func() { temp.Put(1, reflect.TypeOf(0)) })

	temp.Put(999999990, reflect.TypeOf((*rms.MessageExample)(nil)))
	require.Panics(t, func() { temp.Put(999999990, reflect.TypeOf((*rms.MessageExample)(nil))) })
	require.Panics(t, func() { temp.Put(0, reflect.TypeOf((*rms.MessageExample)(nil))) })
	temp.Put(1, reflect.TypeOf((*rms.MessageExample)(nil)))

	temp.PutSpecial(999999990, "Head", reflect.TypeOf((*rms.MessageExample_Head)(nil)))
	temp.PutSpecial(999999990, "HeadX", reflect.TypeOf((*rms.MessageExample_Head)(nil)))
	require.Panics(t, func() { temp.PutSpecial(999999990, "Head", reflect.TypeOf((*rms.MessageExample_Head)(nil))) })
	require.Panics(t, func() { temp.PutSpecial(999999990, "", reflect.TypeOf((*rms.MessageExample_Head)(nil))) })
	require.Panics(t, func() { temp.PutSpecial(0, "", reflect.TypeOf((*rms.MessageExample_Head)(nil))) })

	require.Nil(t, temp.Get(0))
	require.Nil(t, temp.Get(99))
	require.Nil(t, temp.GetSpecial(0, "Head"))
	require.Nil(t, temp.GetSpecial(99, "Head"))

	require.Equal(t, reflect.TypeOf((*rms.MessageExample)(nil)).Elem(), temp.Get(1))
	require.Equal(t, reflect.TypeOf((*rms.MessageExample)(nil)).Elem(), temp.Get(999999990))
	require.Equal(t, reflect.TypeOf((*rms.MessageExample)(nil)).Elem(), temp.GetSpecial(1, ""))
	require.Equal(t, reflect.TypeOf((*rms.MessageExample)(nil)).Elem(), temp.GetSpecial(999999990, ""))

	require.Nil(t, temp.GetSpecial(1, "Head"))
	require.Equal(t, reflect.TypeOf((*rms.MessageExample_Head)(nil)).Elem(), temp.GetSpecial(999999990, "Head"))

	id, m, err := rmsreg.Unmarshal([]byte{0x80, 0x1, 0x1})
	require.NoError(t, err)
	require.Equal(t, uint64(1), id)
	require.Equal(t, &rms.MessageExample{}, m)

	id, m, err = rmsreg.Unmarshal([]byte{0x80, 0x1, 0})
	require.Error(t, err)

	id, m, err = rmsreg.Unmarshal([]byte{0x08, 0x1, 1})
	require.Error(t, err)

	id, m, err = rmsreg.UnmarshalSpecial([]byte{0x80, 0x1, 1}, "Head")
	require.Error(t, err)

	temp.PutSpecial(1, "Head", reflect.TypeOf((*rms.MessageExample_Head)(nil)))

	id, m, err = rmsreg.UnmarshalSpecial([]byte{0x80, 0x1, 1}, "Head")
	require.NoError(t, err)
	require.Equal(t, uint64(1), id)
	require.Equal(t, &rms.MessageExample_Head{}, m)
}
