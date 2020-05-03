// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
	m := MessageExample{MsgParam: 11}
	require.Equal(t, 999999990, int(m.GetDefaultPolymorphID()))
	require.Equal(t, 999999999, int(m.RecordExample.GetDefaultPolymorphID()))

	require.Equal(t, 11, int(m.GetMsgParam()))
	h := m.AsHeadFace()
	require.Equal(t, 11, int(h.GetMsgParam()))
	require.Equal(t, 999999990, int(h.(*MessageExample_Head).GetDefaultPolymorphID()))
}

func TestExampleUnmarshal(t *testing.T) {
	m := &MessageExample{MsgParam: 11, MsgBytes: []byte("abc")}
	m.InitFieldMap(true)
	b, err := m.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, b)

	recordBytes := m.FieldMap.Message
	require.NotEmpty(t, recordBytes)

	// Polymorph == 0 uses default value on serialization
	// so it has to be set explicitly to equal with a deserialized form
	m.RecordExample.Polymorph = uint32(m.RecordExample.GetDefaultPolymorphID())

	id, m2, err := Unmarshal(b)
	require.NoError(t, err)
	require.Equal(t, m.GetDefaultPolymorphID(), id)
	require.True(t, m.Equal(m2))

	id, r2, err := Unmarshal(recordBytes)
	require.NoError(t, err)
	require.Equal(t, m.RecordExample.GetDefaultPolymorphID(), id)
	require.True(t, m.RecordExample.Equal(r2))

	head := m.AsHead()
	b, err = head.Marshal()
	require.NoError(t, err)
	require.NotEmpty(t, b)

	id, m2, err = Unmarshal(b)
	require.NoError(t, err)
	require.Equal(t, m.GetDefaultPolymorphID(), id)

	// head doesn't pass all fields
	m.MsgBytes = nil
	m.RecordExample = RecordExample{Str: m.RecordExample.Str}

	require.Equal(t, m, m2)
}
