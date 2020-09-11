// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package rms

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringer(t *testing.T) {
	v := VCallRequest{}
	require.NotContains(t, v.String(), "PANIC")
}
