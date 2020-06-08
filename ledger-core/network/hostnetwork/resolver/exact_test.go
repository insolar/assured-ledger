// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package resolver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExactResolver(t *testing.T) {
	localAddress := "127.0.0.1:12345"

	r := NewExactResolver()
	require.IsType(t, &exactResolver{}, r)
	realAddress, err := r.Resolve(localAddress)
	require.NoError(t, err)
	require.Equal(t, localAddress, realAddress)
}
