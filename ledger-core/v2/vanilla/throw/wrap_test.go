// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package throw

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

type typedDetails1 struct {
	Value int
}

type typedDetails2 struct {
	Value string
}

func TestAsDetail(t *testing.T) {
	err := E("A", typedDetails1{99})
	err = &net.OpError{Err: err, Op: "test"}
	err = W(err, "B", typedDetails2{"xyz"})

	var data1 typedDetails1
	var data2 typedDetails2
	var data3 net.OpError

	require.True(t, FindDetail(err, &data1))
	require.Equal(t, data1.Value, 99)
	require.True(t, FindDetail(err, &data2))
	require.Equal(t, data2.Value, "xyz")
	require.True(t, FindDetail(err, &data3))
	require.Equal(t, data3.Op, "test")
}
