// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/assured-ledger/blob/master/LICENSE.md.

package jet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenesisExactID(t *testing.T) {
	require.EqualValues(t, 0, GenesisExactID.ID())
	require.EqualValues(t, 0, GenesisExactID.BitLen())
	require.True(t, GenesisExactID.HasLength())
}

func TestUnknownExactID(t *testing.T) {
	require.EqualValues(t, 0, UnknownExactID.ID())
	require.False(t, UnknownExactID.HasLength())
}

func TestExactID(t *testing.T) {
	id := ID(1)
	eid := id.AsExact(7)
	require.EqualValues(t, 1, eid.ID())
	require.EqualValues(t, 7, eid.BitLen())
	require.True(t, eid.HasLength())
}
