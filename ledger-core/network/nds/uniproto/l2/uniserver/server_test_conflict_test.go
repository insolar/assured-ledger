// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package uniserver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdjustServerPort(t *testing.T) {
	const errMsg = "listen udp 127.0.0.1:51077: bind: An attempt was made to access a socket in a way forbidden by its access permissions."
	const localPrefix = "127.0.0.1:"
	const localAddr = localPrefix + "0"

	srv := &UnifiedServer{}

	srv.config.BindingAddress = localPrefix + "9999"
	require.False(t, adjustServerPort(srv, 0, funcNoRand, errMsg))
	require.Equal(t, localPrefix + "9999", srv.config.BindingAddress)

	srv.config.BindingAddress = localAddr

	require.False(t, adjustServerPort(srv, 0, funcNoRand, "another message"))
	require.Equal(t, localAddr, srv.config.BindingAddress)

	for attempt, expected := range []string{
		"26501", "47108", "34886", "47301", "39175",
		"47494", "41416", "47687", "42633", "47880", "43338",
		"2059", "2060", "2061", "2062", "2063",
	} {
		require.True(t, adjustServerPort(srv, attempt, funcNoRand, errMsg), attempt)
		require.Equal(t, localPrefix + expected, srv.config.BindingAddress, attempt)
	}
}

func funcNoRand(int) int {
	return 0
}

