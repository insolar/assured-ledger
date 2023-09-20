package insproto

import (
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
)

func TestFieldMapFQN(t *testing.T) {
	name := proto.MessageName(&FieldMap{})
	require.Equal(t, `.`+name, FieldMapFQN)
	require.Equal(t, FieldMapPackage, reflect.TypeOf(FieldMap{}).PkgPath())
}

func TestFieldMapOps(t *testing.T) {
	for _, m := range []*FieldMap{{Message: []byte{1}}, nil} {
		require.True(t, m.Equal(m))
		require.True(t, m.Equal(&FieldMap{}))
		require.True(t, m.Equal(&FieldMap{Message: []byte{2}}))
		require.True(t, m.Equal(nil))

		require.Error(t, m.Unmarshal(nil))

		n, err := m.MarshalTo(make([]byte, 16))
		require.NoError(t, err)
		require.Zero(t, n)

		n, err = m.MarshalToSizedBuffer(make([]byte, 16))
		require.NoError(t, err)
		require.Zero(t, n)

		require.Nil(t, m.Get(1))
		m.Put(1, 0, 1, []byte{99})
		require.True(t, (m.Get(1) != nil) == (m != nil))
		require.True(t, (m.GetMessage() != nil) == (m != nil))
		m.PutMessage(0, 0, nil)
		require.Nil(t, m.GetMessage())
		m.PutMessage(0, 1, []byte{1})
		m.UnsetMap()
		require.Nil(t, m.GetMessage())
		require.Nil(t, m.Get(1))

		if m == nil {
			continue
		}
		c := &testFieldMapCallback{t, false}
		m.Callback = c
		m.PutMessage(0, 1, []byte{1})
		require.True(t, c.flag)
	}
}

func TestNewFieldMapDescriptorProto(t *testing.T) {
	d := NewFieldMapDescriptorProto(0)
	require.NotNil(t, d)
	require.NotZero(t, d.GetNumber())
	require.Equal(t, FieldMapFieldName, d.GetName())

	d = NewFieldMapDescriptorProto(11)
	require.NotNil(t, d)
	require.Equal(t, int32(11), d.GetNumber())
}

type testFieldMapCallback struct {
	t    *testing.T
	flag bool
}

func (p *testFieldMapCallback) OnMessage(m *FieldMap) {
	require.NotNil(p.t, m)
	require.Equal(p.t, p, m.Callback)
	require.False(p.t, p.flag)
	p.flag = true
}
