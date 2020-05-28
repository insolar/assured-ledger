// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
)

func TestCallFlags_BitStorageIsEngough(t *testing.T) {
	t.Run("interference", func(t *testing.T) {
		require.True(t, contract.InterferenceFlagCount <= 1<<bitInterferenceFlagCount)
	})

	t.Run("state", func(t *testing.T) {
		require.True(t, contract.StateFlagCount <= 1<<bitStateFlagCount)
	})
}

func TestCallFlags(t *testing.T) {
	t.Run("tolerance", func(t *testing.T) {
		flags := CallFlags(0)

		assert.Equal(t, contract.InterferenceFlag(0), flags.GetInterference())

		flags = flags.WithInterference(contract.CallTolerable)

		assert.Equal(t, contract.CallTolerable, flags.GetInterference())
	})

	t.Run("state", func(t *testing.T) {
		flags := CallFlags(0)

		assert.Equal(t, contract.StateFlag(0), flags.GetState())

		flags = flags.WithState(contract.CallValidated)

		assert.Equal(t, CallFlags(8), flags, "%b", flags)

		assert.Equal(t, contract.CallValidated, flags.GetState())

		flags = flags.WithState(contract.CallDirty)

		assert.Equal(t, CallFlags(4), flags, "%b", flags)

		assert.Equal(t, contract.CallDirty, flags.GetState())
	})

	t.Run("mixed", func(t *testing.T) {
		flags := CallFlags(0)

		assert.Equal(t, contract.InterferenceFlag(0), flags.GetInterference())

		assert.Equal(t, contract.StateFlag(0), flags.GetState())

		flags = flags.WithInterference(contract.CallTolerable)

		assert.Equal(t, CallFlags(2), flags, "%b", flags)

		assert.Equal(t, contract.CallTolerable, flags.GetInterference())

		assert.Equal(t, contract.StateFlag(0), flags.GetState())

		flags = flags.WithState(contract.CallValidated)

		assert.Equal(t, CallFlags(10), flags, "%b", flags)

		assert.Equal(t, contract.CallTolerable, flags.GetInterference())

		flags = flags.WithInterference(contract.CallIntolerable)

		assert.Equal(t, CallFlags(9), flags, "%b", flags)

		assert.Equal(t, contract.CallValidated, flags.GetState())

		flags = flags.WithInterference(contract.CallTolerable)

		assert.Equal(t, CallFlags(10), flags, "%b", flags)

		assert.Equal(t, contract.CallTolerable, flags.GetInterference())

		flags = flags.WithState(contract.CallDirty)

		assert.Equal(t, CallFlags(6), flags, "%b", flags)

		assert.Equal(t, contract.CallDirty, flags.GetState())
	})
}
