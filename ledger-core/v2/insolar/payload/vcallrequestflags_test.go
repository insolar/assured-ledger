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

func TestCallRequestFlags_BitStorageIsEngough(t *testing.T) {
	t.Run("interference", func(t *testing.T) {
		require.True(t, contract.InterferenceFlagCount <= 1<<bitInterferenceFlagCount)
	})

	t.Run("state", func(t *testing.T) {
		require.True(t, contract.StateFlagCount <= 1<<bitStateFlagCount)
	})
}

func TestCallRequestFlags(t *testing.T) {
	t.Run("tolerance", func(t *testing.T) {
		flags := CallRequestFlags(0)

		assert.Equal(t, contract.CallIntolerable, flags.GetInterference())

		flags = flags.WithInterference(contract.CallTolerable)

		assert.Equal(t, contract.CallTolerable, flags.GetInterference())
	})

	t.Run("state", func(t *testing.T) {
		flags := CallRequestFlags(0)

		assert.Equal(t, contract.CallDirty, flags.GetState())

		flags = flags.WithState(contract.CallValidated)

		assert.Equal(t, CallRequestFlags(4), flags, "%b", flags)

		assert.Equal(t, contract.CallValidated, flags.GetState())

		flags = flags.WithState(contract.CallDirty)

		assert.Equal(t, CallRequestFlags(0), flags, "%b", flags)

		assert.Equal(t, contract.CallDirty, flags.GetState())
	})

	t.Run("mixed", func(t *testing.T) {
		flags := CallRequestFlags(0)

		assert.Equal(t, contract.CallIntolerable, flags.GetInterference())

		assert.Equal(t, contract.CallDirty, flags.GetState())

		flags = flags.WithInterference(contract.CallTolerable)

		assert.Equal(t, CallRequestFlags(1), flags, "%b", flags)

		assert.Equal(t, contract.CallTolerable, flags.GetInterference())

		assert.Equal(t, contract.CallDirty, flags.GetState())

		flags = flags.WithState(contract.CallValidated)

		assert.Equal(t, CallRequestFlags(5), flags, "%b", flags)

		assert.Equal(t, contract.CallTolerable, flags.GetInterference())

		flags = flags.WithInterference(contract.CallIntolerable)

		assert.Equal(t, CallRequestFlags(4), flags, "%b", flags)

		assert.Equal(t, contract.CallValidated, flags.GetState())

		flags = flags.WithInterference(contract.CallTolerable)

		assert.Equal(t, CallRequestFlags(5), flags, "%b", flags)

		assert.Equal(t, contract.CallTolerable, flags.GetInterference())

		flags = flags.WithState(contract.CallDirty)

		assert.Equal(t, CallRequestFlags(1), flags, "%b", flags)

		assert.Equal(t, contract.CallDirty, flags.GetState())
	})
}
