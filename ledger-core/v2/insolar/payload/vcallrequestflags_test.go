// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCallRequestFlags(t *testing.T) {
	t.Run("tolerance", func(t *testing.T) {
		flags := CallRequestFlags(0)

		assert.Equal(t, CallIntolerable, flags.GetTolerance())

		flags.SetTolerance(CallTolerable)

		assert.Equal(t, CallTolerable, flags.GetTolerance())
	})

	t.Run("state", func(t *testing.T) {
		flags := CallRequestFlags(0)

		assert.Equal(t, CallDirty, flags.GetState())

		flags.SetState(CallValidated)

		assert.Equal(t, CallRequestFlags(4), flags, "%b", flags)

		assert.Equal(t, CallValidated, flags.GetState())

		flags.SetState(CallDirty)

		assert.Equal(t, CallRequestFlags(0), flags, "%b", flags)

		assert.Equal(t, CallDirty, flags.GetState())
	})

	t.Run("mixed", func(t *testing.T) {
		flags := CallRequestFlags(0)

		assert.Equal(t, CallIntolerable, flags.GetTolerance())

		assert.Equal(t, CallDirty, flags.GetState())

		flags.SetTolerance(CallTolerable)

		assert.Equal(t, CallRequestFlags(1), flags, "%b", flags)

		assert.Equal(t, CallTolerable, flags.GetTolerance())

		assert.Equal(t, CallDirty, flags.GetState())

		flags.SetState(CallValidated)

		assert.Equal(t, CallRequestFlags(5), flags, "%b", flags)

		assert.Equal(t, CallTolerable, flags.GetTolerance())

		flags.SetTolerance(CallIntolerable)

		assert.Equal(t, CallRequestFlags(4), flags, "%b", flags)

		assert.Equal(t, CallValidated, flags.GetState())

		flags.SetTolerance(CallTolerable)

		assert.Equal(t, CallRequestFlags(5), flags, "%b", flags)

		assert.Equal(t, CallTolerable, flags.GetTolerance())

		flags.SetState(CallDirty)

		assert.Equal(t, CallRequestFlags(1), flags, "%b", flags)

		assert.Equal(t, CallDirty, flags.GetState())
	})
}
