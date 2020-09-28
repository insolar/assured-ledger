// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package msgdelivery

import (
	"testing"

	"github.com/insolar/assured-ledger/ledger-core/network/nwapi"
	"github.com/stretchr/testify/require"
)

func TestDeliveryAddress(t *testing.T) {
	require.Panics(t, func() {
		NewDirectAddress(0)
	})

	{
		addr := DeliveryAddress{
			addrType:     DirectAddress,
			nodeSelector: uint32(0),
		}
		require.True(t, addr.IsZero())
	}

	require.Panics(t, func() {
		NewRoleAddress(0, 0)
	})

	require.Panics(t, func() {
		add := NewRoleAddress(1, 0)

		_ = add.AsDirect()
	})

	addr := NewDirectAddress(1)

	require.False(t, addr.IsZero())

	addDirect := addr.AsDirect()
	require.Equal(t, nwapi.NewHostID(nwapi.HostID(1)), addDirect)
}
