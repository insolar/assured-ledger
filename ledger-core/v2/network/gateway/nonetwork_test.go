// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package gateway

import (
	"testing"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/v2/network"

	"github.com/stretchr/testify/require"
)

func TestPause(t *testing.T) {
	minTimeout := time.Duration(1)
	maxTimeout := time.Duration(6)
	timeoutMult := time.Duration(2)
	options := network.Options{MinTimeout: minTimeout, MaxTimeout: maxTimeout, TimeoutMult: timeoutMult}
	nn := NoNetwork{Base: &Base{Options: &options}}
	require.Zero(t, nn.pause())

	require.Equal(t, minTimeout, nn.pause())

	require.Equal(t, timeoutMult*minTimeout, nn.pause())

	require.Equal(t, timeoutMult*timeoutMult*minTimeout, nn.pause())

	require.Equal(t, maxTimeout, nn.pause())
}
