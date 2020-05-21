// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/insolar/assured-ledger/ledger-core/v2/insolar/contract"
)

func TestCallRequestFlags(t *testing.T) {
	t.Run("tolerance", func(t *testing.T) {
		flags := CallRequestFlags(0)

		assert.Equal(t, contract.CallIntolerable, flags.GetInterference())

		flags.SetInterference(contract.CallTolerable)

		assert.Equal(t, contract.CallTolerable, flags.GetInterference())
	})

	t.Run("state", func(t *testing.T) {
		flags := CallRequestFlags(0)

		assert.Equal(t, contract.CallDirty, flags.GetState())

		flags.SetState(contract.CallValidated)

		assert.Equal(t, CallRequestFlags(4), flags, "%b", flags)

		assert.Equal(t, contract.CallValidated, flags.GetState())

		flags.SetState(contract.CallDirty)

		assert.Equal(t, CallRequestFlags(0), flags, "%b", flags)

		assert.Equal(t, contract.CallDirty, flags.GetState())
	})

	t.Run("mixed", func(t *testing.T) {
		flags := CallRequestFlags(0)

		assert.Equal(t, contract.CallIntolerable, flags.GetInterference())

		assert.Equal(t, contract.CallDirty, flags.GetState())

		flags.SetInterference(contract.CallTolerable)

		assert.Equal(t, CallRequestFlags(1), flags, "%b", flags)

		assert.Equal(t, contract.CallTolerable, flags.GetInterference())

		assert.Equal(t, contract.CallDirty, flags.GetState())

		flags.SetState(contract.CallValidated)

		assert.Equal(t, CallRequestFlags(5), flags, "%b", flags)

		assert.Equal(t, contract.CallTolerable, flags.GetInterference())

		flags.SetInterference(contract.CallIntolerable)

		assert.Equal(t, CallRequestFlags(4), flags, "%b", flags)

		assert.Equal(t, contract.CallValidated, flags.GetState())

		flags.SetInterference(contract.CallTolerable)

		assert.Equal(t, CallRequestFlags(5), flags, "%b", flags)

		assert.Equal(t, contract.CallTolerable, flags.GetInterference())

		flags.SetState(contract.CallDirty)

		assert.Equal(t, CallRequestFlags(1), flags, "%b", flags)

		assert.Equal(t, contract.CallDirty, flags.GetState())
	})
}
