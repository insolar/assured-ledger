// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/insolar/contract/isolation"
)

func TestCallFlags_BitStorageIsEngough(t *testing.T) {
	t.Run("interference", func(t *testing.T) {
		require.True(t, isolation.InterferenceFlagCount <= 1<<bitInterferenceFlagCount)
	})

	t.Run("state", func(t *testing.T) {
		require.True(t, isolation.StateFlagCount <= 1<<bitStateFlagCount)
	})
}

func TestCallFlags(t *testing.T) {
	t.Run("tolerance", func(t *testing.T) {
		flags := CallFlags(0)

		assert.Equal(t, isolation.InterferenceFlag(0), flags.GetInterference())

		flags = flags.WithInterference(isolation.CallTolerable)

		assert.Equal(t, isolation.CallTolerable, flags.GetInterference())
	})

	t.Run("state", func(t *testing.T) {
		flags := CallFlags(0)

		assert.Equal(t, isolation.StateFlag(0), flags.GetState())

		flags = flags.WithState(isolation.CallValidated)

		assert.Equal(t, CallFlags(8), flags, "%b", flags)

		assert.Equal(t, isolation.CallValidated, flags.GetState())

		flags = flags.WithState(isolation.CallDirty)

		assert.Equal(t, CallFlags(4), flags, "%b", flags)

		assert.Equal(t, isolation.CallDirty, flags.GetState())
	})

	t.Run("mixed", func(t *testing.T) {
		flags := CallFlags(0)

		assert.Equal(t, isolation.InterferenceFlag(0), flags.GetInterference())

		assert.Equal(t, isolation.StateFlag(0), flags.GetState())

		flags = flags.WithInterference(isolation.CallTolerable)

		assert.Equal(t, CallFlags(2), flags, "%b", flags)

		assert.Equal(t, isolation.CallTolerable, flags.GetInterference())

		assert.Equal(t, isolation.StateFlag(0), flags.GetState())

		flags = flags.WithState(isolation.CallValidated)

		assert.Equal(t, CallFlags(10), flags, "%b", flags)

		assert.Equal(t, isolation.CallTolerable, flags.GetInterference())

		flags = flags.WithInterference(isolation.CallIntolerable)

		assert.Equal(t, CallFlags(9), flags, "%b", flags)

		assert.Equal(t, isolation.CallValidated, flags.GetState())

		flags = flags.WithInterference(isolation.CallTolerable)

		assert.Equal(t, CallFlags(10), flags, "%b", flags)

		assert.Equal(t, isolation.CallTolerable, flags.GetInterference())

		flags = flags.WithState(isolation.CallDirty)

		assert.Equal(t, CallFlags(6), flags, "%b", flags)

		assert.Equal(t, isolation.CallDirty, flags.GetState())
	})
}
