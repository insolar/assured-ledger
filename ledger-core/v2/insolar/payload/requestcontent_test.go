// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package payload

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStateRequestContentFlags(t *testing.T) {
	t.Run("single flag", func(t *testing.T) {
		flags := StateRequestContentFlags(0)

		flags.Set(RequestLatestValidatedCode)

		assert.Equal(t, RequestLatestValidatedCode, flags)

		assert.True(t, flags.Contains(RequestLatestValidatedCode))

		assert.False(t, flags.Contains(RequestUnorderedQueue))
	})

	t.Run("two flags one by one", func(t *testing.T) {
		flags := StateRequestContentFlags(0)
		flags.Set(RequestLatestValidatedCode)

		flags.Set(RequestUnorderedQueue)

		assert.Equal(t, RequestLatestValidatedCode|RequestUnorderedQueue, flags)

		assert.True(t, flags.Contains(RequestLatestValidatedCode))

		assert.True(t, flags.Contains(RequestUnorderedQueue))
	})

	t.Run("two flags in one go", func(t *testing.T) {
		flags := StateRequestContentFlags(0)
		flags.Set(RequestLatestValidatedCode, RequestUnorderedQueue)

		assert.Equal(t, RequestLatestValidatedCode|RequestUnorderedQueue, flags)

		assert.True(t, flags.Contains(RequestLatestValidatedCode))

		assert.True(t, flags.Contains(RequestUnorderedQueue))
	})
}
