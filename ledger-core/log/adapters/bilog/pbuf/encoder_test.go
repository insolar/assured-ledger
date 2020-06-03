// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package pbuf

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrependSize(t *testing.T) {
	require.Equal(t, fieldLogEntry.TagSize(), prependFieldSize)
}
