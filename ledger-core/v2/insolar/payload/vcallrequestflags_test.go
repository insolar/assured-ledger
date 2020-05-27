// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallRequestFlags(t *testing.T) {
	require.Equal(t, CallRequestFlags(0x0), BuildCallRequestFlags(SendResultDefault, CallDefault))
	require.Equal(t, CallRequestFlags(0x1), BuildCallRequestFlags(SendResultFull, CallDefault))
	require.Equal(t, CallRequestFlags(0x2), BuildCallRequestFlags(SendResultDefault, RepeatedCall))
	require.Equal(t, CallRequestFlags(0x3), BuildCallRequestFlags(SendResultFull, RepeatedCall))

	assert.Panics(t, func() { BuildCallRequestFlags(bitSendResultFullFlagCount+1, CallDefault) })
	assert.Panics(t, func() { BuildCallRequestFlags(SendResultDefault, 2) })
	assert.Panics(t, func() { BuildCallRequestFlags(bitRepeatedCallFlagCount+1, 2) })

	flags := BuildCallRequestFlags(SendResultDefault, CallDefault)
	flags.WithRepeatedCall(RepeatedCall)
	flags.WithSendResultFull(SendResultFull)
	flags.Equal(CallRequestFlags(0x3))
	flags.Equal(BuildCallRequestFlags(SendResultFull, RepeatedCall))
}
