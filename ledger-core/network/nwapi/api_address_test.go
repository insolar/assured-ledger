// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package nwapi

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

func TestAddress_IsLoopback(t *testing.T) {
	require.True(t, NewHost("127.0.0.1").IsLoopback())
	require.True(t, NewHost("127.1.1.1").IsLoopback())
	require.False(t, NewHost("128.0.0.1").IsLoopback())
	require.True(t, NewHost("::1").IsLoopback())
	require.False(t, NewHost("::2").IsLoopback())
}

func TestAddress_MarshalUnmarshal(t *testing.T) {
	h := NewHost("127.0.0.1:123")

	data := make([]byte, h.ProtoSize())
	size, err := h.MarshalTo(data)
	require.NoError(t, err)

	h2 := Address{}
	err = h2.Unmarshal(data)
	require.NoError(t, err)

	assert.Equal(t, h, h2)
	assert.Equal(t, size, h.ProtoSize())
}

